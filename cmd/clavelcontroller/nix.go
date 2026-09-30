package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// nixEvalJSON runs `nix eval --json <expr>` and returns the raw JSON output.
func nixEvalJSON(ctx context.Context, expr string) (string, error) {
	cmd := exec.CommandContext(ctx, "nix", "eval", "--json", expr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Join(fmt.Errorf("nix eval %s failed", expr), errors.New(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// nixEvalApply evaluates a flake attribute and applies it to the given args.
// Args are embedded as JSON strings in the --apply expression.
func nixEvalApply(ctx context.Context, flakeAttr string, args []string) (string, error) {
	quoted := make([]string, len(args))
	for i, a := range args {
		b, _ := json.Marshal(a)
		quoted[i] = string(b)
	}
	applyExpr := "f: f " + strings.Join(quoted, " ")
	cmd := exec.CommandContext(ctx, "nix", "eval", "--json", flakeAttr, "--apply", applyExpr)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Join(fmt.Errorf("nix eval --apply %s failed", flakeAttr), errors.New(stderr.String()), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func nixJSONString(raw string) (string, error) {
	var s string
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return "", err
	}
	return s, nil
}
