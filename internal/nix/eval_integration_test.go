package nix

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const flakeFixture = `{
  outputs = _: {
    units.server = {
      type = "nixos";
      strategies = [ ];
      profile = "default";
      storePath = "/nix/store/deadbeef-nixos-system-server";
    };
  };
}
`

func TestEvalIntegration(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flake.nix"), []byte(flakeFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Eval(context.Background(), dir+"#units.server")
	if err != nil {
		t.Fatalf("Eval() error = %v", err)
	}
	var decoded struct {
		StorePath string `json:"storePath"`
	}
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("Eval() = %q, not valid JSON: %v", got, err)
	}
	if decoded.StorePath != "/nix/store/deadbeef-nixos-system-server" {
		t.Fatalf("Eval() storePath = %q, want /nix/store/deadbeef-nixos-system-server", decoded.StorePath)
	}
}

func TestEvalIntegrationFailure(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	if _, err := Eval(context.Background(), "/tmp/clavel-does-not-exist-xyz#anything"); err == nil {
		t.Fatal("Eval() error = nil, want failure for missing flake")
	}
}
