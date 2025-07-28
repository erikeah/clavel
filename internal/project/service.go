package project

import (
	"context"
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/jinzhu/copier"
)

type ProjectServiceDeleteOptions struct {
}

type ProjectService struct {
	store ProjectStore
}

func (s *ProjectService) Create(ctx context.Context, data *Project) error {
	err := ValidateProject(data)
	if err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	err = s.store.Create(ctx, data.Name, data)
	if err != nil {
		return err
	}
	return nil
}

func (s *ProjectService) Delete(ctx context.Context, name string) error {
	project, err := s.Show(ctx, name)
	if err != nil {
		return err
	}
	if len(project.Metadata.Finalizers) > 0 {
		nowUTC := time.Now().UTC()
		project.Metadata.DeletionTimestamp = &nowUTC
		return s.Update(ctx, name, project)
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
	current, err := s.store.FindOne(ctx, name)
	if err != nil {
		return err
	}
	if current.Name != data.Name {
		return errors.Join(exceptions.InvalidArguments, errors.New("name cannot be changed"))
	}
	if err := copier.CopyWithOption(current, data, copier.Option{IgnoreEmpty: true}); err != nil {
		return errors.Join(exceptions.InternalFailure, err)
	}

	if err := ValidateProject(current); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	return s.store.Update(ctx, name, current)
}

func (s *ProjectService) Watch(ctx context.Context) (<-chan *Project, <-chan error) {
	return s.store.Watch(ctx)
}

func NewProjectService(repository ProjectStore) *ProjectService {
	return &ProjectService{
		repository,
	}
}
