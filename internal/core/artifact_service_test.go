package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type fakeArtifactStore struct {
	created   map[string]*Artifact
	updated   map[string]*Artifact
	deleted   []string
	stored    *Artifact
	updateErr error
}

func newFakeArtifactStore() *fakeArtifactStore {
	return &fakeArtifactStore{created: map[string]*Artifact{}, updated: map[string]*Artifact{}}
}

func (s *fakeArtifactStore) Create(_ context.Context, key string, data *Artifact) error {
	if _, exists := s.created[key]; exists {
		return exceptions.AlreadyExist
	}
	s.created[key] = data
	return nil
}

func (s *fakeArtifactStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func (s *fakeArtifactStore) FindOne(context.Context, string) (*Artifact, error) {
	if s.stored == nil {
		return nil, exceptions.DoesNotExist
	}
	// A read has to hand out its own copy: read-modify-write updates compare
	// and rewrite the object they were given.
	snapshot := *s.stored
	return &snapshot, nil
}

func (s *fakeArtifactStore) List(context.Context) ([]*Artifact, error) { return nil, nil }

func (s *fakeArtifactStore) Update(_ context.Context, key string, data *Artifact) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated[key] = data
	return nil
}

func (s *fakeArtifactStore) Watch(context.Context, bool) (<-chan ArtifactWatchEvent, <-chan error) {
	return nil, nil
}

func newArtifactService(store ArtifactStore) *ArtifactService {
	return NewArtifactService(store)
}

func TestArtifactCreateAssignsServerIdentity(t *testing.T) {
	store := newFakeArtifactStore()
	service := newArtifactService(store)
	artifact := &Artifact{
		Spec:     ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
		Metadata: Metadata{Name: "server"},
	}
	if err := service.Create(context.Background(), artifact); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	created := store.created["server"]
	if created == nil {
		t.Fatal("nothing was stored under the resource name")
	}
	if created.Metadata.UID == "" {
		t.Fatal("server did not assign a UID")
	}
	if created.APIVersion != ArtifactAPIVersion || created.Kind != ArtifactKind {
		t.Fatalf("TypeMeta = %s/%s, want %s/%s", created.APIVersion, created.Kind, ArtifactAPIVersion, ArtifactKind)
	}
	if created.Metadata.CreationTimestamp == nil {
		t.Fatal("CreationTimestamp was not set")
	}
	if created.Metadata.Generation != 0 {
		t.Fatalf("Generation = %d, want 0 after the first spec merge", created.Metadata.Generation)
	}
	if created.Status.Phase != ArtifactPhaseUnspecified {
		t.Fatalf("Phase = %v, want UNSPECIFIED on create", created.Status.Phase)
	}
}

// The store is what selects where the path is looked up, and the local store is
// a legitimate answer: omitting it must not be an invalid resource.
func TestArtifactCreateAcceptsAnUnsetStore(t *testing.T) {
	store := newFakeArtifactStore()
	service := newArtifactService(store)
	artifact := &Artifact{
		Spec:     ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
		Metadata: Metadata{Name: "server"},
	}
	if err := service.Create(context.Background(), artifact); err != nil {
		t.Fatalf("Create() error = %v, want the local store accepted", err)
	}
}

func TestArtifactCreateRejectsMissingEvalRef(t *testing.T) {
	store := newFakeArtifactStore()
	service := newArtifactService(store)
	artifact := &Artifact{Spec: ArtifactSpecification{}, Metadata: Metadata{Name: "server"}}

	err := service.Create(context.Background(), artifact)
	if err == nil {
		t.Fatal("Create() error = nil, want a missing evalRef rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Create() error = %v, want InvalidArguments", err)
	}
	if len(store.created) != 0 {
		t.Fatalf("store holds %d objects, want 0", len(store.created))
	}
}

func TestArtifactCreateRejectsWrongTypeMeta(t *testing.T) {
	store := newFakeArtifactStore()
	service := newArtifactService(store)
	artifact := &Artifact{
		APIVersion: "other/v1",
		Kind:       "SomethingElse",
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
		Metadata:   Metadata{Name: "server"},
	}
	err := service.Create(context.Background(), artifact)
	if err == nil {
		t.Fatal("Create() error = nil, want wrong TypeMeta rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Create() error = %v, want InvalidArguments", err)
	}
}

func TestArtifactUpdateRejectsRename(t *testing.T) {
	now := time.Now().UTC()
	store := newFakeArtifactStore()
	store.stored = &Artifact{
		APIVersion: ArtifactAPIVersion,
		Kind:       ArtifactKind,
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
		Metadata: Metadata{
			Name:              "server",
			UID:               "uid-1",
			ResourceVersion:   "17",
			CreationTimestamp: &now,
		},
	}

	service := newArtifactService(store)
	err := service.Update(context.Background(), "server", &Artifact{
		Spec:     ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
		Metadata: Metadata{Name: "renamed", ResourceVersion: "17"},
	})
	if err == nil {
		t.Fatal("Update() error = nil, want rename rejected")
	}
	if !errors.Is(err, exceptions.InvalidArguments) {
		t.Fatalf("Update() error = %v, want InvalidArguments", err)
	}
}

func newStoredArtifact(finalizers []string, terminating bool) *Artifact {
	now := time.Now().UTC()
	var deletion *time.Time
	if terminating {
		deletion = &now
	}
	return &Artifact{
		APIVersion: ArtifactAPIVersion,
		Kind:       ArtifactKind,
		Spec:       ArtifactSpecification{StorePath: ArtifactStorePath{EvalRef: "server-eval"}},
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

// Nothing is registered by the controller, so the ordinary case is that an
// artifact goes away in one step.
func TestArtifactDeleteRemovesResourceWithoutFinalizers(t *testing.T) {
	store := newFakeArtifactStore()
	store.stored = newStoredArtifact(nil, false)
	service := newArtifactService(store)

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

func TestArtifactDeleteWaitsOnForeignFinalizers(t *testing.T) {
	store := newFakeArtifactStore()
	store.stored = newStoredArtifact([]string{"example.com/cleanup"}, false)
	service := newArtifactService(store)

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

func TestArtifactDeleteIsIdempotent(t *testing.T) {
	store := newFakeArtifactStore()
	service := newArtifactService(store)

	if err := service.Delete(context.Background(), "server"); err != nil {
		t.Fatalf("Delete() error = %v, want nil for a resource that is gone", err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("store deletes = %v, want none", store.deleted)
	}
}

// A write based on a read that no longer matches the stored revision is a
// conflict: the caller has to re-read, not overwrite.
func TestArtifactUpdateRejectsStaleResourceVersion(t *testing.T) {
	store := newFakeArtifactStore()
	store.stored = newStoredArtifact(nil, false)
	service := newArtifactService(store)

	stale := newStoredArtifact(nil, false)
	stale.Metadata.ResourceVersion = "16"
	stale.Status = ArtifactStatus{Phase: ArtifactPhasePending, ObservedGeneration: 1}

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

func TestArtifactUpdatePropagatesStoreConflict(t *testing.T) {
	store := newFakeArtifactStore()
	store.stored = newStoredArtifact(nil, false)
	store.updateErr = exceptions.Conflict
	service := newArtifactService(store)

	update := newStoredArtifact(nil, false)
	update.Status = ArtifactStatus{Phase: ArtifactPhasePending, ObservedGeneration: 1}

	err := service.Update(context.Background(), "server", update)
	if !errors.Is(err, exceptions.Conflict) {
		t.Fatalf("Update() error = %v, want Conflict", err)
	}
	if len(store.updated) != 0 {
		t.Fatalf("store updates = %d, want 0", len(store.updated))
	}
}
