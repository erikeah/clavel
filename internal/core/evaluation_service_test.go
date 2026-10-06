package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type fakeStore struct {
	created map[string]*Evaluation
	updated map[string]*Evaluation
	stored  *Evaluation
}

func newFakeStore() *fakeStore {
	return &fakeStore{created: map[string]*Evaluation{}, updated: map[string]*Evaluation{}}
}

func (s *fakeStore) Create(_ context.Context, key string, data *Evaluation) error {
	if _, exists := s.created[key]; exists {
		return exceptions.AlreadyExist
	}
	s.created[key] = data
	return nil
}

func (s *fakeStore) Delete(context.Context, string) error { return nil }

func (s *fakeStore) FindOne(context.Context, string) (*Evaluation, error) {
	if s.stored == nil {
		return nil, exceptions.DoesNotExist
	}
	return s.stored, nil
}

func (s *fakeStore) List(context.Context) ([]*Evaluation, error) { return nil, nil }

func (s *fakeStore) Update(_ context.Context, key string, data *Evaluation) error {
	s.updated[key] = data
	return nil
}

func (s *fakeStore) Watch(context.Context) (<-chan *Evaluation, <-chan error) {
	return nil, nil
}

func newService(store EvaluationStore) *EvaluationService {
	return NewEvaluationService(store)
}

func TestCreateAssignsServerIdentity(t *testing.T) {
	store := newFakeStore()
	service := newService(store)
	evaluation := &Evaluation{
		Spec: EvaluationSpecification{Reference: "github:erikeah/clavel?dir=example#server"},
		Metadata: Metadata{
			Name: "server",
		},
	}
	if err := service.Create(context.Background(), evaluation); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	created := store.created["server"]
	if created == nil {
		t.Fatal("nothing was stored under the resource name")
	}
	if created.Metadata.UID == "" {
		t.Fatal("server did not assign a UID")
	}
	if created.APIVersion != EvaluationAPIVersion || created.Kind != EvaluationKind {
		t.Fatalf("TypeMeta = %s/%s, want %s/%s", created.APIVersion, created.Kind, EvaluationAPIVersion, EvaluationKind)
	}
	if created.Metadata.CreationTimestamp == nil {
		t.Fatal("CreationTimestamp was not set")
	}
	if created.Metadata.Generation != 0 {
		t.Fatalf("Generation = %d, want 0 after the first spec merge", created.Metadata.Generation)
	}
	if created.Status.Phase != EvaluationPhaseUnspecified {
		t.Fatalf("Phase = %v, want UNSPECIFIED on create", created.Status.Phase)
	}
}

func TestCreateRejectsClientUID(t *testing.T) {
	store := newFakeStore()
	service := newService(store)
	evaluation := &Evaluation{
		Spec:     EvaluationSpecification{Reference: "ref#server"},
		Metadata: Metadata{Name: "server", UID: "client-supplied"},
	}
	err := service.Create(context.Background(), evaluation)
	if err == nil {
		t.Fatal("Create() error = nil, want client UID rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Create() error = %v, want InvalidArguments", err)
	}
}

func TestCreateRejectsWrongTypeMeta(t *testing.T) {
	store := newFakeStore()
	service := newService(store)
	evaluation := &Evaluation{
		APIVersion: "other/v1",
		Kind:       "SomethingElse",
		Spec:       EvaluationSpecification{Reference: "ref#server"},
		Metadata:   Metadata{Name: "server"},
	}
	err := service.Create(context.Background(), evaluation)
	if err == nil {
		t.Fatal("Create() error = nil, want wrong TypeMeta rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Create() error = %v, want InvalidArguments", err)
	}
	if len(store.created) != 0 {
		t.Fatalf("store holds %d objects, want 0", len(store.created))
	}
}

func TestUpdateRejectsRename(t *testing.T) {
	now := time.Now().UTC()
	store := newFakeStore()
	store.stored = &Evaluation{
		APIVersion: EvaluationAPIVersion,
		Kind:       EvaluationKind,
		Spec:       EvaluationSpecification{Reference: "ref#server"},
		Metadata: Metadata{
			Name:              "server",
			UID:               "uid-1",
			ResourceVersion:   "17",
			CreationTimestamp: &now,
		},
	}

	service := newService(store)
	err := service.Update(context.Background(), "server", &Evaluation{
		Spec:     EvaluationSpecification{Reference: "ref#server"},
		Metadata: Metadata{Name: "renamed", ResourceVersion: "17"},
	})
	if err == nil {
		t.Fatal("Update() error = nil, want rename rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Update() error = %v, want InvalidArguments", err)
	}
}
