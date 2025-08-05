package project

import (
	"context"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

type ProjectService interface {
	Create(ctx context.Context, data *Project) error
	Delete(context.Context, string) error
	List(ctx context.Context) ([]*Project, error)
	Show(context.Context, string) (*Project, error)
}

type projectService struct {
	repository ProjectStore
}

func (s *projectService) Create(ctx context.Context, data *Project) error {
	err := ValidateProject(data)
	if err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	err = s.repository.Create(ctx, data.Name, data)
	if err != nil {
		return err
	}
	return nil
}

func (s *projectService) Delete(ctx context.Context, name string) error {
	return s.repository.Delete(ctx, name)
}

func (s *projectService) List(ctx context.Context) ([]*Project, error) {
	return s.repository.List(ctx)
}

func (s *projectService) Show(ctx context.Context, name string) (*Project, error) {
	return s.repository.FindOne(ctx, name)
}

func NewProjectService(repository ProjectStore) *projectService {
	return &projectService{
		repository,
	}
}
