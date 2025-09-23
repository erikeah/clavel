package project

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type ProjectService struct {
	store       ProjectStore
	setDefaults func(*Project) error
	merge       func(over *Project, from *Project) error
	validate    func(*Project) error
}

func (s *ProjectService) Create(ctx context.Context, data *Project) error {
	resource := &Project{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if err := s.merge(resource, data); err != nil {
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

func (s *ProjectService) Delete(ctx context.Context, name string) error {
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

func (s *ProjectService) List(ctx context.Context) ([]*Project, error) {
	return s.store.List(ctx)
}

func (s *ProjectService) Show(ctx context.Context, name string) (*Project, error) {
	return s.store.FindOne(ctx, name)
}

func (s *ProjectService) Update(ctx context.Context, name string, data *Project) error {
	target, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if err := s.merge(target, data); err != nil {
		return errors.Join(exceptions.InternalFailure, err)
	}
	if err := s.validate(target); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	return s.store.Update(ctx, name, target)
}

func (s *ProjectService) Watch(ctx context.Context) (<-chan *Project, <-chan error) {
	return s.store.Watch(ctx)
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{
		store:       store,
		validate:    ValidateProject,
		merge:       MergeProject,
		setDefaults: SetDefaults_Project,
	}
}
