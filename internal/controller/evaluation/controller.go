package evaluation

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/controller"
	"github.com/erikeah/clavel/internal/core"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

const (
	apiAddressEnv  = "CLAVEL_API_ADDR"
	defaultAPIAddr = "http://localhost:8080"
	workerCountEnv = "CLAVEL_CONTROLLER_WORKERS"
	defaultWorkers = 4
	watchReconnect = time.Second
)

func apiAddress() string {
	if address := os.Getenv(apiAddressEnv); address != "" {
		return address
	}
	return defaultAPIAddr
}

func workerCount() int {
	if raw := os.Getenv(workerCountEnv); raw != "" {
		if count, err := strconv.Atoi(raw); err == nil && count > 0 {
			return count
		}
	}
	return defaultWorkers
}

// startInformer pumps the watch stream into informer.Add, reconnecting on
// stream errors. Initial sync comes from the server via the List request flag.
func startInformer(ctx context.Context, informer *controller.Informer[*core.Evaluation]) {
	client := corev1connect.NewEvaluationServiceClient(http.DefaultClient, apiAddress())
	watchRequest := &connect.Request[corev1.EvaluationServiceWatchRequest]{
		Msg: &corev1.EvaluationServiceWatchRequest{List: true},
	}
	for {
		watchResponse, err := client.Watch(ctx, watchRequest)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("watch failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(watchReconnect):
			}
			continue
		}
		for watchResponse.Receive() {
			if data := watchResponse.Msg().GetData(); data != nil {
				informer.Add(data.Convert(nil))
			}
			if streamError := watchResponse.Msg().GetError(); streamError != nil {
				slog.Error(streamError.GetMessage())
			}
		}
		if err := watchResponse.Err(); err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("watch stream ended", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchReconnect):
		}
	}
}

func newReconcile(informer *controller.Informer[*core.Evaluation]) controller.ReconcileFunc {
	return func(ctx context.Context, key string) error {
		evaluation, ok := informer.Get(key)
		if !ok {
			// Resource deleted before reconcile; cleanup/finalizers are a follow-up.
			slog.Info("evaluation deleted", "name", key)
			return nil
		}
		slog.Info("evaluation reconciled", "name", evaluation.Name, "reference", evaluation.Spec.Reference)
		return nil
	}
}

// Run starts the evaluation controller and blocks until ctx is cancelled.
func Run(ctx context.Context) {
	queue := controller.NewWorkQueue()
	informer := controller.NewInformer[*core.Evaluation](queue, func(evaluation *core.Evaluation) string {
		return evaluation.Name
	})
	go startInformer(ctx, informer)
	controller.NewController(informer, newReconcile(informer), workerCount()).Run(ctx)
}
