package core

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EvaluationWatchEvent is one notification from the store's watch stream: a
// resource that exists (Put), one that was removed (Delete), or the end of the
// initial snapshot (Sync).
type EvaluationWatchEvent = genericstore.Event[*Evaluation]

type EvaluationStore interface {
	Create(ctx context.Context, key string, data *Evaluation) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*Evaluation, error)
	List(ctx context.Context) ([]*Evaluation, error)
	Update(ctx context.Context, key string, data *Evaluation) error
	// Watch streams store events; snapshot replays the current state first,
	// terminated by a Sync event, and resumes gap-free from that revision.
	Watch(ctx context.Context, snapshot bool) (<-chan EvaluationWatchEvent, <-chan error)
}

func NewEvaluationStore(cli *clientv3.Client) EvaluationStore {
	return genericstore.NewStore[*Evaluation](cli, []string{"evaluations"})
}
