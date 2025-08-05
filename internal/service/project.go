package service

import (
	"context"
	"errors"

	"github.com/erikeah/clavel/internal/exception"
	"github.com/erikeah/clavel/internal/model"
	"github.com/erikeah/clavel/internal/repository"
	"github.com/erikeah/clavel/internal/validation"
)

type ProjectService interface {
	List(ctx context.Context) ([]*model.Project, error)
	Show(context.Context, string) (*model.Project, error)
	Add(ctx context.Context, data *model.Project) error
}

type projectService struct {
	repository repository.ProjectRepository
}

func (s *projectService) Show(ctx context.Context, name string) (*model.Project, error) {
	return s.repository.FindOne(ctx, name)
}

func (s *projectService) Add(ctx context.Context, data *model.Project) error {
	err := validation.ValidateProject(data)
	if err != nil {
		return errors.Join(exception.InvalidArguments, err)
	}
	err = s.repository.Add(ctx, data.Name, data)
	if err != nil {
		return err
	}
	return nil
}

func (s *projectService) List(ctx context.Context) ([]*model.Project, error) {
	return s.repository.List(ctx)
}

func NewProjectService(repository repository.ProjectRepository) *projectService {
	return &projectService{
		repository,
	}
}
