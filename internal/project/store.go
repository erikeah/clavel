package project

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ProjectStore interface {
	List(ctx context.Context) ([]*Project, error)
	FindOne(context.Context, string) (*Project, error)
	Delete(context.Context, string) error
	Create(ctx context.Context, key string, data *Project) error
}

func NewProjectStore(cli *clientv3.Client) ProjectStore {
	projectRepository := genericstore.NewStore[Project](cli, []string{"projects"})
	return projectRepository
}
