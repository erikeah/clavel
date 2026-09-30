package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/erikeah/clavel/internal/core"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
)

func (c *controller) reconcile(ctx context.Context, j job) error {
	resource := j.resource
	if resource.GetMetadata().GetDeletionTimestamp() != "" {
		return nil
	}
	crd := j.crd
	slog.Info("reconciling resource",
		"kind", kindKey(crd.GetGroup(), crd.GetVersion(), crd.GetPlural()),
		"name", resource.GetName())

	evaluationName, err := core.SpecStringField(
		crd.GetSchema(),
		crd.GetSpecMessage(),
		resource.GetSpec(),
		"evaluation",
	)
	if err != nil {
		return fmt.Errorf("resolve referenced evaluation: %w", err)
	}

	evaluationResp, err := c.evalClient.Show(ctx, connect.NewRequest(&corev1.EvaluationServiceShowRequest{Name: evaluationName}))
	if err != nil {
		return fmt.Errorf("show evaluation %s: %w", evaluationName, err)
	}
	evaluation := evaluationResp.Msg.GetData()
	reference := evaluation.GetSpec().GetReference()

	// If already deployed for this derivation, skip re-deploying.
	if status := evaluation.GetStatus(); status != nil {
		if status.GetPhase() == corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_READY &&
			status.GetResult() != nil && status.GetResult().GetStorePath() != "" {
			existing, _, err := c.evalDerivation(ctx, crd, reference)
			if err == nil && existing == status.GetResult().GetStorePath() {
				slog.Info("already deployed", "evaluation", evaluationName, "store_path", existing)
				return nil
			}
		}
	}

	derivation, hash, err := c.evalDerivation(ctx, crd, reference)
	if err != nil {
		c.setEvaluationStatus(ctx, evaluationName, corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_FAILED, nil)
		return err
	}
	if err := c.setEvaluationStatus(ctx, evaluationName, corev1.EvaluationStatusPhase_EVALUATION_STATUS_PHASE_READY,
		&corev1.EvaluationStatusResult{Hash: hash, StorePath: derivation}); err != nil {
		return err
	}

	action, err := c.invokeAction(ctx, crd, string(resource.GetSpec()), reference, derivation)
	if err != nil {
		return err
	}
	slog.Info("executing action", "evaluation", evaluationName, "type", action.Type)
	if err := c.executeAction(action); err != nil {
		return err
	}
	slog.Info("deployed", "evaluation", evaluationName, "store_path", derivation)
	return nil
}

// evalDerivation invokes the action module's `reference` to transform the
// evaluation reference, evaluates it, and returns the store path and hash.
func (c *controller) evalDerivation(ctx context.Context, crd *corev1.CustomResourceDefinition, reference string) (string, string, error) {
	actionModule := crd.GetActionModule()
	if actionModule == "" {
		return "", "", fmt.Errorf("CRD has no action module")
	}
	transformed, err := nixEvalApply(ctx, actionModule+".reference", []string{reference})
	if err != nil {
		return "", "", err
	}
	ref, err := nixJSONString(transformed)
	if err != nil {
		return "", "", fmt.Errorf("parse transformed reference: %w", err)
	}
	raw, err := nixEvalJSON(ctx, ref)
	if err != nil {
		return "", "", err
	}
	derivation, err := nixJSONString(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse derivation value: %w", err)
	}
	return derivation, filepath.Base(derivation), nil
}

// invokeAction calls the action module's `apply` and parses the deployment action.
func (c *controller) invokeAction(ctx context.Context, crd *corev1.CustomResourceDefinition, spec string, reference string, derivation string) (deploymentAction, error) {
	var action deploymentAction
	actionModule := crd.GetActionModule()
	raw, err := nixEvalApply(ctx, actionModule+".apply", []string{spec, reference, derivation})
	if err != nil {
		return action, err
	}
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		return action, fmt.Errorf("parse deployment action: %w", err)
	}
	return action, nil
}

func (c *controller) setEvaluationStatus(ctx context.Context, name string, phase corev1.EvaluationStatusPhase, result *corev1.EvaluationStatusResult) error {
	resp, err := c.evalClient.Show(ctx, connect.NewRequest(&corev1.EvaluationServiceShowRequest{Name: name}))
	if err != nil {
		return err
	}
	current := resp.Msg.GetData()
	current.Status = &corev1.EvaluationStatus{
		Phase:  phase,
		Result: result,
	}
	_, err = c.evalClient.Update(ctx, connect.NewRequest(&corev1.EvaluationServiceUpdateRequest{
		Name: name,
		Data: current,
	}))
	return err
}
