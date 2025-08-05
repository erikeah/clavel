package repository

import (
	"context"

	"github.com/erikeah/clavel/internal/errors"
	"github.com/erikeah/clavel/internal/model"
)

type ProjectRepository interface {
	FindOne(context.Context, string) (*model.Project, errors.Error)
	Add(ctx context.Context, key string, data *model.Project) errors.Error
}

func NewProjectRepository() ProjectRepository {
	projectRepository := NewRepository[model.Project]([]string{"projects"})
	return projectRepository
}
