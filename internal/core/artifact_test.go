package core

import (
	"strings"
	"testing"
)

func TestStorePathFromResult(t *testing.T) {
	for name, testcase := range map[string]struct {
		result    string
		want      string
		wantError bool
	}{
		"valid": {
			result: EncodeResult([]byte(`{"storePath":"/nix/store/abc123-nixos-system-server"}`)),
			want:   "/nix/store/abc123-nixos-system-server",
		},
		"empty result":    {result: "", wantError: true},
		"not base64":      {result: "not base64!", wantError: true},
		"not json":        {result: EncodeResult([]byte("plain")), wantError: true},
		"not an object":   {result: EncodeResult([]byte(`"/nix/store/abc"`)), wantError: true},
		"no storePath":    {result: EncodeResult([]byte(`{"other":true}`)), wantError: true},
		"outside a store": {result: EncodeResult([]byte(`{"storePath":"/tmp/abc"}`)), wantError: true},
		"truncated store": {result: EncodeResult([]byte(`{"storePath":"/nix/"}`)), wantError: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := StorePathFromResult(testcase.result)
			if testcase.wantError {
				if err == nil {
					t.Fatalf("StorePathFromResult() = %q, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("StorePathFromResult() error = %v", err)
			}
			if got != testcase.want {
				t.Fatalf("StorePathFromResult() = %q, want %q", got, testcase.want)
			}
		})
	}
}

// An artifact is a reference: which store to look in is optional, but the
// evaluation naming the path never is.
func TestValidateArtifactSpecificationAllowsAnUnsetStore(t *testing.T) {
	spec := ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server"}}
	if err := ValidateArtifactSpecification(spec); err != nil {
		t.Fatalf("ValidateArtifactSpecification() error = %v, want nil for the local store", err)
	}
}

func TestValidateArtifactSpecificationRequiresEvalRef(t *testing.T) {
	for name, spec := range map[string]ArtifactSpecification{
		"missing":   {},
		"has slash": {StorePath: ArtifactStorePath{EvalRef: "team/server"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateArtifactSpecification(spec); err == nil {
				t.Fatal("ValidateArtifactSpecification() error = nil, want a rejection")
			}
		})
	}
}

func TestValidateArtifactAcceptsAnUnsetStore(t *testing.T) {
	artifact := &Artifact{
		APIVersion: ArtifactAPIVersion,
		Kind:       ArtifactKind,
		Metadata:   Metadata{Name: "server"},
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
	}
	if err := SetDefaults_Artifact(artifact); err != nil {
		t.Fatalf("SetDefaults_Artifact() error = %v", err)
	}
	if err := ValidateArtifact(*artifact); err != nil {
		t.Fatalf("ValidateArtifact() error = %v", err)
	}
}

// Retargeting the reference is a spec change, so it has to make the resource
// observe a new generation.
func TestMergeArtifactBumpsGenerationWhenTheEvalRefChanges(t *testing.T) {
	stored := &Artifact{
		APIVersion: ArtifactAPIVersion,
		Kind:       ArtifactKind,
		Metadata:   Metadata{Name: "server", Generation: 3},
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "old-eval"}},
	}
	incoming := &Artifact{
		APIVersion: ArtifactAPIVersion,
		Kind:       ArtifactKind,
		Metadata:   Metadata{Name: "server"},
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "new-eval"}},
	}

	changed, err := MergeArtifact(stored, incoming)
	if err != nil {
		t.Fatalf("MergeArtifact() error = %v", err)
	}
	if !changed {
		t.Fatal("MergeArtifact() = false, want the retarget reported as a change")
	}
	if stored.Metadata.Generation != 4 {
		t.Fatalf("Generation = %d, want 4", stored.Metadata.Generation)
	}
	if stored.Spec.StorePath.EvalRef != "new-eval" {
		t.Fatalf("evalRef = %q, want new-eval", stored.Spec.StorePath.EvalRef)
	}
}

func TestMergeArtifactSpecificationCarriesTheStore(t *testing.T) {
	over := ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}}
	from := ArtifactSpecification{Store: "https://cache.nixos.org", StorePath: ArtifactStorePath{EvalRef: "server-eval"}}

	changed, err := MergeArtifactSpecification(&over, &from)
	if err != nil {
		t.Fatalf("MergeArtifactSpecification() error = %v", err)
	}
	if !changed || over.Store != "https://cache.nixos.org" {
		t.Fatalf("changed = %v store = %q, want the store applied", changed, over.Store)
	}
}

func TestMergeArtifactStatusIsReplaceOnly(t *testing.T) {
	over := ArtifactStatus{Phase: ArtifactPhasePending, ObservedGeneration: 1, Message: "waiting"}
	from := ArtifactStatus{Phase: ArtifactPhaseSucceeded, StorePath: "/nix/store/abc", ObservedGeneration: 1}

	changed, err := MergeArtifactStatus(&over, from)
	if err != nil {
		t.Fatalf("MergeArtifactStatus() error = %v", err)
	}
	if !changed || over != from {
		t.Fatalf("changed = %v status = %+v, want the incoming status", changed, over)
	}

	changed, err = MergeArtifactStatus(&over, from)
	if err != nil {
		t.Fatalf("MergeArtifactStatus() error = %v", err)
	}
	if changed {
		t.Fatal("MergeArtifactStatus() = true for an identical status")
	}
}

func TestValidateArtifactRejectsWrongTypeMeta(t *testing.T) {
	err := ValidateArtifactTypeMeta("other/v1", "SomethingElse")
	if err == nil {
		t.Fatal("ValidateArtifactTypeMeta() error = nil, want a rejection")
	}
	if !strings.Contains(err.Error(), ArtifactAPIVersion) {
		t.Fatalf("error = %v, want the expected apiVersion", err)
	}
}

func TestSetDefaultsArtifactFillsTypeMeta(t *testing.T) {
	artifact := &Artifact{}
	if err := SetDefaults_Artifact(artifact); err != nil {
		t.Fatalf("SetDefaults_Artifact() error = %v", err)
	}
	if artifact.APIVersion != ArtifactAPIVersion || artifact.Kind != ArtifactKind {
		t.Fatalf("TypeMeta = %s/%s, want %s/%s", artifact.APIVersion, artifact.Kind, ArtifactAPIVersion, ArtifactKind)
	}
	if err := SetDefaults_Artifact(nil); err == nil {
		t.Fatal("SetDefaults_Artifact(nil) error = nil, want a failure")
	}
}
