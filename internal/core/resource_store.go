package core

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ResourceStore interface {
	Create(ctx context.Context, name string, data *Resource) error
	Delete(ctx context.Context, name string) error
	FindOne(ctx context.Context, name string) (*Resource, error)
	List(ctx context.Context) ([]*Resource, error)
	Update(ctx context.Context, name string, data *Resource) error
	Watch(ctx context.Context) (<-chan *Resource, <-chan error)
}

type ResourceStoreProvider func(group string, version string, plural string) ResourceStore

func NewResourceStoreProvider(cli *clientv3.Client) ResourceStoreProvider {
	return func(group string, version string, plural string) ResourceStore {
		return genericstore.NewStore[Resource](cli, []string{"resource", group, version, plural})
	}
}

func NewResourceStore(cli *clientv3.Client, group string, version string, plural string) ResourceStore {
	return genericstore.NewStore[Resource](cli, []string{"resource", group, version, plural})
}
