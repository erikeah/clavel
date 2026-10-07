package nix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

// PathInfo reports whether storePath exists in the store selected by store; an
// empty store means the local store. It queries rather than materialises: the
// path is a reference into a store someone else owns, and clavel only records
// whether that reference resolves.
//
// A path the store does not have is not an error. nix path-info exits 0 and
// reports it as null in info, so presence has to come from parsing the reply,
// not from the exit status; a non-zero exit is reserved for a real failure such
// as an unreachable store.
func PathInfo(ctx context.Context, store string, storePath string) (bool, error) {
	args := []string{"path-info", "--json", "--json-format", "2"}
	if store != "" {
		args = append(args, "--store", store)
	}
	args = append(args, storePath)
	cmd := exec.CommandContext(ctx, "nix", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false, errors.Join(err, errors.New(strings.TrimSpace(stderr.String())))
	}
	return hasPathInfo(stdout.String(), storePath)
}

// hasPathInfo parses the format 2 reply, whose info is keyed by store-path
// basename and whose value is null for a path the queried store does not have.
func hasPathInfo(stdout string, storePath string) (bool, error) {
	var reply struct {
		Info map[string]json.RawMessage `json:"info"`
	}
	if err := json.Unmarshal([]byte(stdout), &reply); err != nil {
		return false, errors.Join(err, errors.New("nix path-info produced invalid JSON"))
	}
	value, ok := reply.Info[filepath.Base(storePath)]
	return ok && !bytes.Equal(value, []byte("null")), nil
}
