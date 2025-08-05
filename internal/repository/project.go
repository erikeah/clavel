package repository

import (
	"context"

	"github.com/erikeah/clavel/internal/model"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ProjectRepository interface {
	List(ctx context.Context) ([]*model.Project, error)
	FindOne(context.Context, string) (*model.Project, error)
	Add(ctx context.Context, key string, data *model.Project) error
}

func NewProjectRepository(cli *clientv3.Client) ProjectRepository {
	projectRepository := NewRepository[model.Project](cli, []string{"projects"})
	return projectRepository
}
