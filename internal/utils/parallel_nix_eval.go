package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"sync"
)

type nixEvaluationResult struct {
	Value any   `json:"value"`
	Err   error `json:"error,omitempty"`
}

func eval(ctx context.Context, evaluationResult any, flakeRef string, extraArgs ...string) error {
	args := []string{"eval"}
	args = append(args, flakeRef)
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	args = append(args, "--eval-cache")
	args = append(args, "--json")
	cmd := exec.CommandContext(ctx, "nix", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		err = errors.Join(errors.New(stderr.String()), err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		err = errors.Join(errors.New(stderr.String()), err)
		return err
	}
	if err := json.Unmarshal(stdout.Bytes(), &evaluationResult); err != nil {
		err = errors.Join(errors.New(stderr.String()), err)
		return err
	}
	return nil
}

func ParallelNixEval(ctx context.Context, references []string) <-chan *nixEvaluationResult {
	var results = make(chan *nixEvaluationResult, len(references))
	var wg sync.WaitGroup
	for _, reference := range references {
		wg.Add(1)
		go func(ref string) {
			defer wg.Done()
			var value any
			if err := eval(ctx, &value, ref); err != nil {
				results <- &nixEvaluationResult{Value: nil, Err: err}
				return
			}
			results <- &nixEvaluationResult{Value: value}
		}(reference)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
