package projectapi

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/project"
)

type ProjectService struct {
	store       ProjectStore
	setDefaults func(*project.Project) error
	merge       func(over *project.Project, from *project.Project) (bool, error)
	validate    func(*project.Project) error
}

func (s *ProjectService) Create(ctx context.Context, data *project.Project) error {
	resource := &project.Project{}
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

func (s *ProjectService) List(ctx context.Context) ([]*project.Project, error) {
	return s.store.List(ctx)
}

func (s *ProjectService) Show(ctx context.Context, name string) (*project.Project, error) {
	return s.store.FindOne(ctx, name)
}

func (s *ProjectService) Update(ctx context.Context, name string, data *project.Project) error {
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

func (s *ProjectService) Watch(ctx context.Context) (<-chan *project.Project, <-chan error) {
	return s.store.Watch(ctx)
}

func NewProjectService(store ProjectStore) *ProjectService {
	return &ProjectService{
		store:       store,
		validate:    project.ValidateProject,
		merge:       project.MergeProject,
		setDefaults: project.SetDefaults_Project,
	}
}
