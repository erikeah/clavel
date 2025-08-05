package service

import (
	"context"
	"errors"

	"github.com/erikeah/clavel/internal/exception"
	"github.com/erikeah/clavel/internal/model"
	"github.com/erikeah/clavel/internal/repository"
	"github.com/erikeah/clavel/internal/validation"
	"github.com/jinzhu/copier"
)

type ProjectService interface {
	Show(context.Context, string) (*model.Project, error)
	Add(ctx context.Context, data *model.Project) error
}

type projectService struct {
	repository repository.ProjectRepository
}

func (s *projectService) Show(ctx context.Context, name string) (*model.Project, error) {
	value, err := s.repository.FindOne(ctx, name)
	if err != nil {
		return nil, err
	}
	project := &model.Project{}
	if err := copier.Copy(project, value); err != nil {
		return nil, err
	}
	return project, nil
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

func NewProjectService(repository repository.ProjectRepository) *projectService {
	return &projectService{
		repository,
	}
}
