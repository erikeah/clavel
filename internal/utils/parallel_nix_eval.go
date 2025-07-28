package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func ParallelNixEval(ctx context.Context, flakeRef string) (chan *nixEvaluationResult, error) {
	var attrNames []string
	err := eval(ctx, &attrNames, flakeRef, "--apply", "builtins.attrNames")
	if err != nil {
		return nil, err
	}
	var wg sync.WaitGroup
	var results = make(chan *nixEvaluationResult, len(attrNames))
	for _, attrName := range attrNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			var value any
			applyFunc := fmt.Sprintf("arg: arg.%s", name)
			err := eval(ctx, &value, flakeRef, "--apply", applyFunc)
			if err != nil {
				results <- &nixEvaluationResult{Value: nil, Err: err}
			}
			results <- &nixEvaluationResult{Value: value}
		}(attrName)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return results, nil
}
