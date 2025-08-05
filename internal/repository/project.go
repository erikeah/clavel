package repository

import (
	"context"

	"github.com/erikeah/clavel/internal/model"
)

type ProjectRepository interface {
	FindOne(context.Context, string) (*model.Project, error)
	Add(ctx context.Context, key string, data *model.Project) error
}

func NewProjectRepository() ProjectRepository {
	projectRepository := NewRepository[model.Project]([]string{"projects"})
	return projectRepository
}
