package core

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EvaluationStore interface {
	Create(ctx context.Context, key string, data *Evaluation) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*Evaluation, error)
	List(ctx context.Context) ([]*Evaluation, error)
	Update(ctx context.Context, key string, data *Evaluation) error
	Watch(context.Context) (<-chan *Evaluation, <-chan error)
}

func NewEvaluationStore(cli *clientv3.Client) EvaluationStore {
	return genericstore.NewStore[Evaluation](cli, []string{"evaluations"})
}
