package core

import (
	"context"

	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// ArtifactWatchEvent is one notification from the store's watch stream: a
// resource that exists (Put), one that was removed (Delete), or the end of the
// initial snapshot (Sync).
type ArtifactWatchEvent = genericstore.Event[*Artifact]

type ArtifactStore interface {
	Create(ctx context.Context, key string, data *Artifact) error
	Delete(context.Context, string) error
	FindOne(context.Context, string) (*Artifact, error)
	List(ctx context.Context) ([]*Artifact, error)
	Update(ctx context.Context, key string, data *Artifact) error
	// Watch streams store events; snapshot replays the current state first,
	// terminated by a Sync event, and resumes gap-free from that revision.
	Watch(ctx context.Context, snapshot bool) (<-chan ArtifactWatchEvent, <-chan error)
}

func NewArtifactStore(cli *clientv3.Client) ArtifactStore {
	return genericstore.NewStore[*Artifact](cli, []string{"artifacts"})
}
