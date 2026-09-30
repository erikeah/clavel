package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"

	"connectrpc.com/connect"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
	"github.com/erikeah/clavel/pkg/api/core/v1/corev1connect"
)

type descriptor struct {
	CustomResourceDefinitions []struct {
		Group        string          `json:"group"`
		Version      string          `json:"version"`
		Kind         string          `json:"kind"`
		Plural       string          `json:"plural"`
		Schema       string          `json:"schema"`
		SpecMessage  string          `json:"specMessage"`
		ActionModule string          `json:"actionModule"`
		Metadata     json.RawMessage `json:"metadata"`
	} `json:"customResourceDefinitions"`
	Evaluations []struct {
		Name     string          `json:"name"`
		Metadata json.RawMessage `json:"metadata"`
		Spec     map[string]any  `json:"spec"`
	} `json:"evaluations"`
	Resources []struct {
		Group    string          `json:"group"`
		Version  string          `json:"version"`
		Plural   string          `json:"plural"`
		Kind     string          `json:"kind"`
		Name     string          `json:"name"`
		Metadata json.RawMessage `json:"metadata"`
		Spec     string          `json:"spec"`
	} `json:"resources"`
}

func evalConfiguration(flakeRef string, config string) (*descriptor, error) {
	target := flakeRef + "#clavelConfigurations." + config
	cmd := exec.Command("nix", "eval", target, "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, errors.Join(fmt.Errorf("nix eval %s failed", target), errors.New(stderr.String()), err)
	}
	var d descriptor
	if err := json.Unmarshal(stdout.Bytes(), &d); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to parse apply descriptor"), err)
	}
	return &d, nil
}

func apply(flakeRef string, config string, address string) error {
	d, err := evalConfiguration(flakeRef, config)
	if err != nil {
		return err
	}
	ctx := context.Background()
	httpClient := http.DefaultClient

	crdClient := corev1connect.NewCustomResourceDefinitionServiceClient(httpClient, address)
	for _, c := range d.CustomResourceDefinitions {
		schema, err := base64.StdEncoding.DecodeString(c.Schema)
		if err != nil {
			return errors.Join(fmt.Errorf("CRD %s: invalid schema base64", c.Kind), err)
		}
		crd := &corev1.CustomResourceDefinition{
			Group:        c.Group,
			Version:      c.Version,
			Kind:         c.Kind,
			Plural:       c.Plural,
			Schema:       schema,
			SpecMessage:  c.SpecMessage,
			ActionModule: c.ActionModule,
			Metadata:     &corev1.Metadata{},
		}
		if _, err := crdClient.Create(ctx, connect.NewRequest(&corev1.CustomResourceDefinitionServiceCreateRequest{Data: crd})); err != nil {
			fmt.Printf("crd %s/%s: %v\n", c.Kind, c.Version, err)
		} else {
			fmt.Printf("crd %s/%s: applied\n", c.Kind, c.Version)
		}
	}

	evalClient := corev1connect.NewEvaluationServiceClient(httpClient, address)
	for _, e := range d.Evaluations {
		reference, _ := e.Spec["reference"].(string)
		ev := &corev1.Evaluation{
			Name:     e.Name,
			Metadata: &corev1.Metadata{},
			Spec:     &corev1.EvaluationSpecification{Reference: reference},
		}
		if _, err := evalClient.Create(ctx, connect.NewRequest(&corev1.EvaluationServiceCreateRequest{Data: ev})); err != nil {
			fmt.Printf("evaluation %s: %v\n", e.Name, err)
		} else {
			fmt.Printf("evaluation %s: applied\n", e.Name)
		}
	}

	resourceClient := corev1connect.NewCustomResourceServiceClient(httpClient, address)
	for _, r := range d.Resources {
		resource := &corev1.Resource{
			Group:    r.Group,
			Version:  r.Version,
			Plural:   r.Plural,
			Kind:     r.Kind,
			Name:     r.Name,
			Metadata: &corev1.Metadata{},
			Spec:     []byte(r.Spec),
		}
		if _, err := resourceClient.Create(ctx, connect.NewRequest(&corev1.CustomResourceServiceCreateRequest{Data: resource})); err != nil {
			fmt.Printf("resource %s.%s/%s: %v\n", r.Plural, r.Group, r.Name, err)
		} else {
			fmt.Printf("resource %s.%s/%s: applied\n", r.Plural, r.Group, r.Name)
		}
	}
	return nil
}
