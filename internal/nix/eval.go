package nix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
)

// Eval evaluates a flake reference and returns the raw JSON result.
func Eval(ctx context.Context, reference string) (string, error) {
	cmd := exec.CommandContext(ctx, "nix", "eval", "--eval-cache", "--json", reference)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.Join(err, errors.New(stderr.String()))
	}
	result := strings.TrimSpace(stdout.String())
	if !json.Valid([]byte(result)) {
		return "", errors.New("nix eval produced invalid JSON")
	}
	return result, nil
}
