package nix

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A format 2 reply is keyed by store-path basename, and the value is what tells
// presence apart from absence.
func TestHasPathInfo(t *testing.T) {
	const storePath = "/nix/store/abc123def456-nixos-system-server"
	for name, testcase := range map[string]struct {
		stdout    string
		want      bool
		wantError bool
	}{
		"present": {
			stdout: `{"info":{"abc123def456-nixos-system-server":{"narHash":"sha256-1"},"narSize":12},"storeDir":"/nix/store","version":2}`,
			want:   true,
		},
		"absent": {
			stdout: `{"info":{"abc123def456-nixos-system-server":null},"storeDir":"/nix/store","version":2}`,
			want:   false,
		},
		"not mentioned at all": {
			stdout: `{"info":{},"storeDir":"/nix/store","version":2}`,
			want:   false,
		},
		"no info key": {
			stdout: `{"storeDir":"/nix/store","version":2}`,
			want:   false,
		},
		// Format 1 keys by full path, so a basename lookup must not match it.
		"full path key": {
			stdout: `{"/nix/store/abc123def456-nixos-system-server":{}}`,
			want:   false,
		},
		"invalid json": {
			stdout:    "not json",
			wantError: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := hasPathInfo(testcase.stdout, storePath)
			if testcase.wantError {
				if err == nil {
					t.Fatal("hasPathInfo() error = nil, want a parse failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("hasPathInfo() error = %v", err)
			}
			if got != testcase.want {
				t.Fatalf("hasPathInfo() = %v, want %v", got, testcase.want)
			}
		})
	}
}

// existingStorePath returns a path that is actually in the local store, so the
// integration test asks nix about something present rather than only about
// something absent.
func existingStorePath(t *testing.T) string {
	t.Helper()
	entries, err := os.ReadDir("/nix/store")
	if err != nil {
		t.Skipf("local store unavailable: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		// Store objects are named <32 char hash>-<name>; dotfiles such as
		// .links and .trash are not store paths.
		if len(name) < 34 || name[0] == '.' || name[32] != '-' {
			continue
		}
		return filepath.Join("/nix/store", name)
	}
	t.Skip("local store holds no regular store paths")
	return ""
}

func TestPathInfoReportsPresenceInTheLocalStore(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	present := existingStorePath(t)

	got, err := PathInfo(context.Background(), "", present)
	if err != nil {
		t.Fatalf("PathInfo(%q) error = %v", present, err)
	}
	if !got {
		t.Fatalf("PathInfo(%q) = false, want true for a path in the local store", present)
	}
}

func TestPathInfoReportsAnAbsentPathWithoutFailing(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	// nix path-info exits 0 and answers null for a path it does not have, so
	// absence has to be a boolean rather than an error.
	absent := "/nix/store/00000000000000000000000000000000-clavel-absent"

	got, err := PathInfo(context.Background(), "", absent)
	if err != nil {
		t.Fatalf("PathInfo(%q) error = %v, want nil for an absent path", absent, err)
	}
	if got {
		t.Fatal("PathInfo() = true, want false for a path the store does not have")
	}
}

func TestPathInfoFailsWhenTheStoreCannotBeReached(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	// A scheme nix cannot open is a query failure, not an absent path: only a
	// successful reply decides presence.
	if _, err := PathInfo(context.Background(), "clavel+bogus://", "/nix/store/00000000000000000000000000000000-x"); err == nil {
		t.Fatal("PathInfo() error = nil, want a failure for an unusable store")
	}
}

func TestPathInfoQueriesTheGivenStore(t *testing.T) {
	if _, err := exec.LookPath("nix"); err != nil {
		t.Skip("nix not available")
	}
	// A non-empty store has to reach nix as --store; "auto" is the local store
	// named explicitly, which is what the empty spec falls back to.
	present := existingStorePath(t)
	got, err := PathInfo(context.Background(), "auto", present)
	if err != nil {
		t.Fatalf("PathInfo() error = %v", err)
	}
	if !got {
		t.Fatalf("PathInfo(%q) = false, want true", present)
	}
}
