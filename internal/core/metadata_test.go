package core

import (
	"errors"
	"testing"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

func TestSetDefaultsMetadataAssignsUID(t *testing.T) {
	meta := &Metadata{}
	if err := SetDefaults_Metadata(meta); err != nil {
		t.Fatalf("SetDefaults_Metadata() error = %v", err)
	}
	if meta.UID == "" {
		t.Fatal("UID was not assigned")
	}
	if meta.Generation != -1 {
		t.Fatalf("Generation = %d, want -1", meta.Generation)
	}
	if meta.CreationTimestamp == nil {
		t.Fatal("CreationTimestamp was not set")
	}
	uid := meta.UID
	if err := SetDefaults_Metadata(meta); err != nil {
		t.Fatalf("SetDefaults_Metadata() re-run error = %v", err)
	}
	if meta.UID != uid {
		t.Fatalf("UID changed on re-run: %q -> %q", uid, meta.UID)
	}
}

func TestMergeMetadataRejectsUIDChange(t *testing.T) {
	over := &Metadata{UID: "assigned-uid"}
	from := &Metadata{UID: "client-uid"}
	if _, err := MergeMetadata(over, from); err == nil {
		t.Fatal("MergeMetadata() error = nil, want UID change rejected")
	} else if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("MergeMetadata() error = %v, want InvalidArguments", err)
	}
	if over.UID != "assigned-uid" {
		t.Fatalf("UID = %q, want it untouched", over.UID)
	}
}

func TestMergeMetadataRejectsRename(t *testing.T) {
	over := &Metadata{Name: "server"}
	from := &Metadata{Name: "renamed"}
	if _, err := MergeMetadata(over, from); err == nil {
		t.Fatal("MergeMetadata() error = nil, want rename rejected")
	}
}

func TestMergeMetadataSetsNameWhenUnset(t *testing.T) {
	over := &Metadata{UID: "assigned-uid"}
	from := &Metadata{Name: "server"}
	changed, err := MergeMetadata(over, from)
	if err != nil {
		t.Fatalf("MergeMetadata() error = %v", err)
	}
	if !changed {
		t.Fatal("MergeMetadata() changed = false, want true")
	}
	if over.Name != "server" {
		t.Fatalf("Name = %q, want server", over.Name)
	}
}

func TestMergeMetadataKeepsUIDWhenAbsent(t *testing.T) {
	over := &Metadata{UID: "assigned-uid"}
	from := &Metadata{}
	changed, err := MergeMetadata(over, from)
	if err != nil {
		t.Fatalf("MergeMetadata() error = %v", err)
	}
	if changed {
		t.Fatal("MergeMetadata() changed = true, want false for empty patch")
	}
	if over.UID != "assigned-uid" {
		t.Fatalf("UID = %q, want assigned-uid", over.UID)
	}
}

func TestMergeMetadataResourceVersionMismatch(t *testing.T) {
	over := &Metadata{ResourceVersion: "17"}
	from := &Metadata{ResourceVersion: "18"}
	_, err := MergeMetadata(over, from)
	if err == nil {
		t.Fatal("MergeMetadata() error = nil, want resourceVersion conflict")
	}
	if !errors.Is(err, exceptions.Conflict) {
		t.Fatalf("MergeMetadata() error = %v, want Conflict", err)
	}
}

func TestMergeMetadataReplacesMapsWhenProvided(t *testing.T) {
	over := &Metadata{
		Labels:      map[string]string{"old": "value"},
		Annotations: map[string]string{"old": "value"},
	}
	from := &Metadata{
		Labels:      map[string]string{"app": "server"},
		Annotations: map[string]string{"note": "hello"},
	}
	changed, err := MergeMetadata(over, from)
	if err != nil {
		t.Fatalf("MergeMetadata() error = %v", err)
	}
	if !changed {
		t.Fatal("MergeMetadata() changed = false, want true")
	}
	if len(over.Labels) != 1 || over.Labels["app"] != "server" {
		t.Fatalf("Labels = %v, want replaced", over.Labels)
	}
	if len(over.Annotations) != 1 || over.Annotations["note"] != "hello" {
		t.Fatalf("Annotations = %v, want replaced", over.Annotations)
	}
}

func TestMergeMetadataKeepsMapsWhenAbsent(t *testing.T) {
	over := &Metadata{Labels: map[string]string{"keep": "me"}}
	from := &Metadata{}
	changed, err := MergeMetadata(over, from)
	if err != nil {
		t.Fatalf("MergeMetadata() error = %v", err)
	}
	if changed {
		t.Fatal("MergeMetadata() changed = true, want false")
	}
	if over.Labels["keep"] != "me" {
		t.Fatalf("Labels = %v, want kept", over.Labels)
	}
}

func TestMergeMetadataRejectsNamespaceChange(t *testing.T) {
	over := &Metadata{Namespace: "production"}
	from := &Metadata{Namespace: "staging"}
	if _, err := MergeMetadata(over, from); err == nil {
		t.Fatal("MergeMetadata() error = nil, want namespace change rejected")
	}
}

func TestValidateMetadata(t *testing.T) {
	now := time.Now().UTC()
	valid := Metadata{
		Name:              "server",
		UID:               "uid-1",
		CreationTimestamp: &now,
		Labels:            map[string]string{"app.kubernetes.io/name": "server"},
		Annotations:       map[string]string{"clavel.io/note": "hi"},
		OwnerReferences: []OwnerReference{{
			APIVersion: EvaluationAPIVersion,
			Kind:       EvaluationKind,
			Name:       "root",
			UID:        "uid-root",
			Controller: true,
		}},
	}
	if err := ValidateMetadata(&valid); err != nil {
		t.Fatalf("ValidateMetadata() error = %v, want nil", err)
	}

	cases := map[string]func(m *Metadata){
		"empty name":        func(m *Metadata) { m.Name = "" },
		"name with slash":   func(m *Metadata) { m.Name = "a/b" },
		"missing uid":       func(m *Metadata) { m.UID = "" },
		"missing timestamp": func(m *Metadata) { m.CreationTimestamp = nil },
		"bad label key":     func(m *Metadata) { m.Labels = map[string]string{"bad key": "v"} },
		"bad label value":   func(m *Metadata) { m.Labels = map[string]string{"key": "bad value"} },
		"empty owner ref":   func(m *Metadata) { m.OwnerReferences = []OwnerReference{{Name: "root"}} },
		"two controllers": func(m *Metadata) {
			m.OwnerReferences = []OwnerReference{{APIVersion: "v", Kind: "K", Name: "a", UID: "1", Controller: true}, {APIVersion: "v", Kind: "K", Name: "b", UID: "2", Controller: true}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			metadata := valid
			metadata.Labels = map[string]string{"app.kubernetes.io/name": "server"}
			metadata.OwnerReferences = []OwnerReference{valid.OwnerReferences[0]}
			mutate(&metadata)
			if err := ValidateMetadata(&metadata); err == nil {
				t.Fatalf("ValidateMetadata() error = nil, want error for %s", name)
			}
		})
	}
}
