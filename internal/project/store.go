package project

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ProjectStore interface {
	Create(ctx context.Context, key string, data *Project) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	Update(ctx context.Context, key string, data *Project) error
	Watch(context.Context) (<-chan *Project, <-chan error)
}

func NewProjectStore(cli *clientv3.Client) ProjectStore {
	projectRepository := genericstore.NewStore[Project](cli, []string{"projects"})
	return projectRepository
}
