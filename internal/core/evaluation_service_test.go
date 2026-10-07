package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type fakeStore struct {
	created   map[string]*Evaluation
	updated   map[string]*Evaluation
	deleted   []string
	stored    *Evaluation
	updateErr error
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

func (s *fakeStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *fakeStore) FindOne(context.Context, string) (*Evaluation, error) {
	if s.stored == nil {
		return nil, exceptions.DoesNotExist
	}
	// A read has to hand out its own copy: read-modify-write updates compare
	// and rewrite the object they were given.
	snapshot := *s.stored
	return &snapshot, nil
}

func (s *fakeStore) List(context.Context) ([]*Evaluation, error) { return nil, nil }

func (s *fakeStore) Update(_ context.Context, key string, data *Evaluation) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated[key] = data
	return nil
}

func (s *fakeStore) Watch(context.Context, bool) (<-chan EvaluationWatchEvent, <-chan error) {
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

func newStoredEvaluation(finalizers []string, terminating bool) *Evaluation {
	now := time.Now().UTC()
	var deletion *time.Time
	if terminating {
		deletion = &now
	}
	return &Evaluation{
		APIVersion: EvaluationAPIVersion,
		Kind:       EvaluationKind,
		Spec:       EvaluationSpecification{Reference: "ref#server"},
		Metadata: Metadata{
			Name:              "server",
			UID:               "uid-1",
			ResourceVersion:   "17",
			Finalizers:        finalizers,
			CreationTimestamp: &now,
			DeletionTimestamp: deletion,
		},
	}
}

func TestDeleteRemovesResourceWithoutFinalizers(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation(nil, false)
	service := newService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(store.deleted) != 1 {
		t.Fatalf("store deletes = %v, want [server]", store.deleted)
	}
	if len(store.updated) != 0 {
		t.Fatalf("store updates = %d, want 0", len(store.updated))
	}
}

func TestDeleteCompletesDrainedResource(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation(nil, true)
	service := newService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(store.deleted) != 1 {
		t.Fatalf("store deletes = %v, want [server]", store.deleted)
	}
}

func TestDeleteWaitsOnFinalizers(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation([]string{"clavel.core/evaluation"}, false)
	service := newService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("store deletes = %v, want none while a finalizer blocks", store.deleted)
	}
	terminating := store.updated["server"]
	if terminating == nil {
		t.Fatal("Update() was not called to mark the resource terminating")
	}
	if terminating.Metadata.DeletionTimestamp == nil {
		t.Fatal("deletionTimestamp was not set")
	}
	if len(terminating.Metadata.Finalizers) != 1 {
		t.Fatalf("finalizers = %v, want them preserved", terminating.Metadata.Finalizers)
	}
}

func TestDeleteIsNoOpWhileTerminating(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation([]string{"clavel.core/evaluation"}, true)
	service := newService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(store.deleted) != 0 || len(store.updated) != 0 {
		t.Fatalf("deletes = %v updates = %d, want neither", store.deleted, len(store.updated))
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	store := newFakeStore()
	service := newService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v, want nil for a resource that is gone", err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("store deletes = %v, want none", store.deleted)
	}
}

func TestUpdateKeepsDeletingToReconcile(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation(nil, true)
	service := newService(store)

	update := newStoredEvaluation(nil, true)
	update.Status = EvaluationStatus{Phase: EvaluationPhasePending, ObservedGeneration: 1}
	if err := service.Update(context.Background(), "server", update); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("store deletes = %v, want none: Update must stay a pure write", store.deleted)
	}
	if len(store.updated) != 1 {
		t.Fatalf("store updates = %d, want 1", len(store.updated))
	}
}

// A write based on a read that no longer matches the stored revision is a
// conflict: the caller has to re-read, not overwrite.
func TestUpdateRejectsStaleResourceVersion(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation(nil, false)
	service := newService(store)

	stale := newStoredEvaluation(nil, false)
	stale.Metadata.ResourceVersion = "16"
	stale.Status = EvaluationStatus{Phase: EvaluationPhasePending, ObservedGeneration: 1}

	err := service.Update(context.Background(), "server", stale)
	if err == nil {
		t.Fatal("Update() error = nil, want Conflict for a stale resourceVersion")
	}
	if !errors.Is(err, exceptions.Conflict) {
		t.Fatalf("Update() error = %v, want Conflict", err)
	}
	if len(store.updated) != 0 {
		t.Fatalf("store updates = %d, want 0: a conflict must not reach the store", len(store.updated))
	}
}

func TestUpdatePropagatesStoreConflict(t *testing.T) {
	store := newFakeStore()
	store.stored = newStoredEvaluation(nil, false)
	store.updateErr = exceptions.Conflict
	service := newService(store)

	update := newStoredEvaluation(nil, false)
	update.Status = EvaluationStatus{Phase: EvaluationPhasePending, ObservedGeneration: 1}

	err := service.Update(context.Background(), "server", update)
	if !errors.Is(err, exceptions.Conflict) {
		t.Fatalf("Update() error = %v, want Conflict", err)
	}
	if len(store.updated) != 0 {
		t.Fatalf("store updates = %d, want 0", len(store.updated))
	}
}
