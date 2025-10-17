package projectapi

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	"github.com/erikeah/clavel/internal/project"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ProjectStore interface {
	Create(ctx context.Context, key string, data *project.Project) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*project.Project, error)
	List(ctx context.Context) ([]*project.Project, error)
	Update(ctx context.Context, key string, data *project.Project) error
	Watch(context.Context) (<-chan *project.Project, <-chan error)
}

func NewProjectStore(cli *clientv3.Client) ProjectStore {
	return genericstore.NewStore[project.Project](cli, []string{"projects"})
}
