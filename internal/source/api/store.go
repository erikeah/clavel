package sourceapi

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	"github.com/erikeah/clavel/internal/source"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type SourceStore interface {
	Create(ctx context.Context, key string, data *source.Source) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*source.Source, error)
	List(ctx context.Context) ([]*source.Source, error)
	Update(ctx context.Context, key string, data *source.Source) error
	Watch(context.Context) (<-chan *source.Source, <-chan error)
}

func NewSourceStore(cli *clientv3.Client) SourceStore {
	return genericstore.NewStore[source.Source](cli, []string{"sources"})
}
