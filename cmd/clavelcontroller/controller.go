package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"connectrpc.com/connect"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type controller struct {
	address string

	crdClient      corev1connect.CustomResourceDefinitionServiceClient
	resourceClient corev1connect.CustomResourceServiceClient
	evalClient     corev1connect.EvaluationServiceClient

	mu       sync.Mutex
	kinds    map[string]*corev1.CustomResourceDefinition
	watchers map[string]context.CancelFunc

	jobs chan job
}

type job struct {
	crd      *corev1.CustomResourceDefinition
	resource *corev1.Resource
}

func kindKey(group, version, plural string) string {
	return group + "/" + version + "/" + plural
}

func newController() *controller {
	address := os.Getenv("CLAVEL_API_URL")
	if address == "" {
		address = "http://localhost:8080"
	}
	httpClient := http.DefaultClient
	return &controller{
		address:        address,
		crdClient:      corev1connect.NewCustomResourceDefinitionServiceClient(httpClient, address),
		resourceClient: corev1connect.NewCustomResourceServiceClient(httpClient, address),
		evalClient:     corev1connect.NewEvaluationServiceClient(httpClient, address),
		kinds:          map[string]*corev1.CustomResourceDefinition{},
		watchers:       map[string]context.CancelFunc{},
		jobs:           make(chan job, 256),
	}
}

func (c *controller) Start() {
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		go c.worker(ctx)
	}
	if err := c.watchCRDs(ctx); err != nil {
		slog.Error("crd watch failed", "error", err)
		os.Exit(1)
	}
}

func (c *controller) watchCRDs(ctx context.Context) error {
	watchResp, err := c.crdClient.Watch(ctx, connect.NewRequest(&corev1.CustomResourceDefinitionServiceWatchRequest{List: true}))
	if err != nil {
		return err
	}
	defer watchResp.Close()
	for {
		if !watchResp.Receive() {
			if watchResp.Err() != nil {
				return watchResp.Err()
			}
			return nil
		}
		if data := watchResp.Msg().GetData(); data != nil {
			c.ensureKind(ctx, data)
		}
		if err := watchResp.Msg().GetError(); err != nil {
			slog.Error("crd watch error", "message", err.GetMessage())
		}
	}
}

func (c *controller) ensureKind(ctx context.Context, crd *corev1.CustomResourceDefinition) {
	key := kindKey(crd.GetGroup(), crd.GetVersion(), crd.GetPlural())
	c.mu.Lock()
	existing, ok := c.kinds[key]
	c.kinds[key] = crd
	cancel, hasWatcher := c.watchers[key]
	c.mu.Unlock()

	if ok && existing != nil && !crdChanged(existing, crd) && hasWatcher {
		return
	}
	if hasWatcher && cancel != nil {
		cancel()
	}
	kindCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.watchers[key] = cancel
	c.mu.Unlock()
	go c.watchKind(kindCtx, crd)
}

func crdChanged(a, b *corev1.CustomResourceDefinition) bool {
	return a.GetActionModule() != b.GetActionModule() ||
		a.GetSpecMessage() != b.GetSpecMessage() ||
		string(a.GetSchema()) != string(b.GetSchema())
}

func (c *controller) watchKind(ctx context.Context, crd *corev1.CustomResourceDefinition) {
	key := kindKey(crd.GetGroup(), crd.GetVersion(), crd.GetPlural())
	slog.Info("watching kind", "kind", key)
	watchResp, err := c.resourceClient.Watch(ctx, connect.NewRequest(&corev1.CustomResourceServiceWatchRequest{
		Group:   crd.GetGroup(),
		Version: crd.GetVersion(),
		Plural:  crd.GetPlural(),
		List:    true,
	}))
	if err != nil {
		slog.Error("resource watch failed", "kind", key, "error", err)
		return
	}
	defer watchResp.Close()
	for {
		if !watchResp.Receive() {
			if ctx.Err() != nil {
				return
			}
			slog.Error("resource watch closed", "kind", key, "error", watchResp.Err())
			return
		}
		if data := watchResp.Msg().GetData(); data != nil {
			select {
			case c.jobs <- job{crd: crd, resource: data}:
			case <-ctx.Done():
				return
			}
		}
		if errMsg := watchResp.Msg().GetError(); errMsg != nil {
			slog.Error("resource watch error", "kind", key, "message", errMsg.GetMessage())
		}
	}
}

func (c *controller) worker(ctx context.Context) {
	for {
		select {
		case j := <-c.jobs:
			if err := c.reconcile(ctx, j); err != nil {
				slog.Error("reconcile failed",
					"kind", kindKey(j.crd.GetGroup(), j.crd.GetVersion(), j.crd.GetPlural()),
					"resource", j.resource.GetName(),
					"error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

type deploymentAction struct {
	Type    string   `json:"type"`
	Command []string `json:"command"`
	Host    string   `json:"host"`
}

func (c *controller) executeAction(action deploymentAction) error {
	switch action.Type {
	case "ssh":
		if action.Host == "" {
			return fmt.Errorf("ssh action requires a host")
		}
		args := append([]string{action.Host}, action.Command...)
		out, err := exec.Command("ssh", args...).CombinedOutput()
		if err != nil {
			return errors.Join(fmt.Errorf("ssh command failed: %s", strings.TrimSpace(string(out))), err)
		}
		slog.Info("ssh action ok", "host", action.Host, "out", strings.TrimSpace(string(out)))
		return nil
	case "command", "":
		if len(action.Command) == 0 {
			return fmt.Errorf("command action requires a command")
		}
		out, err := exec.Command(action.Command[0], action.Command[1:]...).CombinedOutput()
		if err != nil {
			return errors.Join(fmt.Errorf("command failed: %s", strings.TrimSpace(string(out))), err)
		}
		slog.Info("command action ok", "command", strings.Join(action.Command, " "), "out", strings.TrimSpace(string(out)))
		return nil
	default:
		return fmt.Errorf("unknown action type %q", action.Type)
	}
}
