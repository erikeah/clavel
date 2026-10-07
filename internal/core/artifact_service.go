package core

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type ArtifactService struct {
	store       ArtifactStore
	setDefaults func(*Artifact) error
	merge       func(over *Artifact, from *Artifact) (bool, error)
	validate    func(Artifact) error
}

func (s *ArtifactService) Create(ctx context.Context, data *Artifact) error {
	resource := &Artifact{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if _, err := s.merge(resource, data); err != nil {
		return err
	}
	resource.Status = ArtifactStatus{}
	if err := s.validate(*resource); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	if err := s.store.Create(ctx, resource.Metadata.Name, resource); err != nil {
		return err
	}
	return nil
}

// Delete removes a resource. Without finalizers it is gone immediately and
// repeated calls are a no-op; with finalizers the resource only gets a
// deletionTimestamp and stays terminating until the last finalizer is dropped,
// at which point a further Delete (typically from reconcile) completes it.
func (s *ArtifactService) Delete(ctx context.Context, name string) error {
	target, err := s.Show(ctx, name)
	if err != nil {
		if errors.Is(err, exceptions.DoesNotExist) {
			return nil
		}
		return err
	}
	if len(target.Metadata.Finalizers) == 0 {
		return s.store.Delete(ctx, name)
	}
	if target.Metadata.DeletionTimestamp != nil {
		return nil
	}
	nowUTC := time.Now().UTC()
	target.Metadata.DeletionTimestamp = &nowUTC
	return s.Update(ctx, name, target)
}

func (s *ArtifactService) List(ctx context.Context) ([]*Artifact, error) {
	return s.store.List(ctx)
}

func (s *ArtifactService) Show(ctx context.Context, name string) (*Artifact, error) {
	return s.store.FindOne(ctx, name)
}

func (s *ArtifactService) Update(ctx context.Context, name string, data *Artifact) error {
	if data == nil {
		return exceptions.InvalidArguments
	}
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if hasChanged, err := s.merge(target, data); err != nil {
		// A conflict is a normal, retryable outcome: it is not an internal
		// failure and must not be reported as one.
		if errors.Is(err, exceptions.Conflict) {
			return err
		}
		return errors.Join(exceptions.InternalFailure, err)
	} else if !hasChanged {
		return nil
	}
	if err := s.validate(*target); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	return s.store.Update(ctx, name, target)
}

// Watch forwards the store's event stream; see ArtifactStore.Watch.
func (s *ArtifactService) Watch(ctx context.Context, snapshot bool) (<-chan ArtifactWatchEvent, <-chan error) {
	return s.store.Watch(ctx, snapshot)
}

func NewArtifactService(store ArtifactStore) *ArtifactService {
	return &ArtifactService{
		store:       store,
		validate:    ValidateArtifact,
		merge:       MergeArtifact,
		setDefaults: SetDefaults_Artifact,
	}
}
