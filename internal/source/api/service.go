package sourceapi

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/source"
)

type SourceService struct {
	store       SourceStore
	setDefaults func(*source.Source) error
	merge       func(over *source.Source, from *source.Source) (bool, error)
	validate    func(*source.Source) error
}

func (s *SourceService) Create(ctx context.Context, data *source.Source) error {
	resource := &source.Source{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if _, err := s.merge(resource, data); err != nil {
		return err
	}
	if err := s.validate(resource); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	if err := s.store.Create(ctx, resource.Name, resource); err != nil {
		return err
	}
	return nil
}

func (s *SourceService) Delete(ctx context.Context, name string) error {
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if len(target.Metadata.Finalizers) > 0 {
		nowUTC := time.Now().UTC()
		target.Metadata.DeletionTimestamp = &nowUTC
		return s.Update(ctx, name, target)
	} else {
		return s.store.Delete(ctx, name)
	}
}

func (s *SourceService) List(ctx context.Context) ([]*source.Source, error) {
	return s.store.List(ctx)
}

func (s *SourceService) Show(ctx context.Context, name string) (*source.Source, error) {
	return s.store.FindOne(ctx, name)
}

func (s *SourceService) Update(ctx context.Context, name string, data *source.Source) error {
	if data == nil {
		return exceptions.InvalidArguments
	}
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if hasChanged, err := s.merge(target, data); err != nil {
		return errors.Join(exceptions.InternalFailure, err)
	} else if !hasChanged {
		return exceptions.NotModified
	}
	if err := s.validate(target); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	return s.store.Update(ctx, name, target)
}

func (s *SourceService) Watch(ctx context.Context) (<-chan *source.Source, <-chan error) {
	return s.store.Watch(ctx)
}

func NewSourceService(store SourceStore) *SourceService {
	return &SourceService{
		store:       store,
		validate:    source.ValidateSource,
		merge:       source.MergeSource,
		setDefaults: source.SetDefaults_Source,
	}
}
