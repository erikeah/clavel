package genericstore

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/erikeah/clavel/internal/exceptions"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type StorableResource interface {
	GetMetadataResourceVersion() string
	SetMetadataResourceVersion(string)
}

// EventType mirrors the watch semantics of the event vocabulary shared with
// the API layer.
type EventType int

const (
	// EventTypePut is a resource that exists now: a snapshot item or a create.
	EventTypePut EventType = iota
	// EventTypeDelete is a resource that was removed. Object is the zero value.
	EventTypeDelete
	// EventTypeSync marks the end of the initial snapshot.
	EventTypeSync
)

// Event is a single watch notification. Name is always the resource identity,
// derived from the store key, so a delete can be handled without an object.
type Event[M any] struct {
	Type   EventType
	Name   string
	Object M
}

type store[M StorableResource] struct {
	client *clientv3.Client
	path   []string
}

func (s *store[M]) genPath(name string) string {
	return s.prefix() + name
}

// prefix is the etcd key space of this store, always slash-terminated so a
// resource name can be recovered from any key under it.
func (s *store[M]) prefix() string {
	return "/" + strings.Join(s.path, "/") + "/"
}

func (s *store[M]) FindOne(ctx context.Context, name string) (M, error) {
	kv := s.client.KV
	resp, err := kv.Get(ctx, s.genPath(name))
	if err != nil {
		var zero M
		return zero, errors.Join(exceptions.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		var zero M
		return zero, exceptions.DoesNotExist
	}
	var model M
	if err := json.Unmarshal(resp.Kvs[0].Value, &model); err != nil {
		var zero M
		return zero, errors.Join(exceptions.Unknown, err)
	}
	model.SetMetadataResourceVersion(strconv.FormatInt(resp.Kvs[0].ModRevision, 10))
	return model, nil
}

func (s *store[M]) List(ctx context.Context) ([]M, error) {
	kv := s.client.KV
	resp, err := kv.Get(ctx, s.genPath(""), clientv3.WithPrefix())
	if err != nil {
		return nil, errors.Join(exceptions.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		return []M{}, nil
	}
	var list []M
	for i, value := range resp.Kvs {
		var model M
		if err := json.Unmarshal(value.Value, &model); err != nil {
			return nil, errors.Join(exceptions.Unknown, err)
		}
		model.SetMetadataResourceVersion(strconv.FormatInt(resp.Kvs[i].ModRevision, 10))
		list = append(list, model)
	}
	return list, nil
}

func (s *store[M]) Create(ctx context.Context, name string, data M) error {
	kv := s.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	destination := s.genPath(name)
	resp, err := kv.
		Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(destination), "=", 0)).
		Then(clientv3.OpPut(destination, string(jsonData))).
		Commit()
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	if !resp.Succeeded {
		return errors.Join(exceptions.AlreadyExist, err)
	}
	return nil
}

func (s *store[M]) Delete(ctx context.Context, name string) error {
	kv := s.client.KV
	resp, err := kv.Delete(ctx, s.genPath(name))
	if err != nil {
		return errors.Join(exceptions.ExternalFailure, err)
	}
	if resp.Deleted == 0 {
		return exceptions.DoesNotExist
	}
	return nil
}

// Update persists data only if the resource still carries the revision the
// caller read: the compare-and-swap rejects a stale write instead of letting
// it clobber a concurrent one. The expected revision is the resource's own
// resourceVersion, which equals the key's ModRevision, so an update that skips
// the read (empty) only ever matches an absent key. On success the stored
// revision is stamped back onto data.
func (s *store[M]) Update(ctx context.Context, key string, data M) error {
	kv := s.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	destination := s.genPath(key)
	expected, err := strconv.ParseInt(data.GetMetadataResourceVersion(), 10, 64)
	if err != nil {
		return errors.Join(exceptions.Conflict, err)
	}
	resp, err := kv.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(destination), "=", expected)).
		Then(clientv3.OpPut(destination, string(jsonData))).
		Else(clientv3.OpGet(destination)).
		Commit()
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	if resp.Succeeded {
		data.SetMetadataResourceVersion(strconv.FormatInt(resp.Header.Revision, 10))
		return nil
	}
	// The precondition failed: either the resource is gone, so the caller's
	// read is stale, or someone wrote it first.
	if len(resp.Responses) < 1 || len(resp.Responses[0].GetResponseRange().GetKvs()) < 1 {
		return exceptions.DoesNotExist
	}
	return errors.Join(exceptions.Conflict, errors.New("resourceVersion does not match"))
}

// Watch streams the state of the store. It always starts at the revision it
// just read, so no change can slip in between; with snapshot it additionally
// replays every existing resource as a put followed by an EventTypeSync, which
// is how a consumer can tell what a stream did not cover. It returns the events
// and a stream error channel; both are closed when ctx is cancelled.
func (s *store[M]) Watch(ctx context.Context, snapshot bool) (<-chan Event[M], <-chan error) {
	ch := make(chan Event[M])
	errCh := make(chan error, 1) // Buffered channel for errors

	go func() {
		defer close(ch)
		defer close(errCh)
		// A blocked consumer must not outlive its ctx: every send is interruptible.
		emit := func(event Event[M]) bool {
			select {
			case ch <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}
		fail := func(err error) bool {
			select {
			case errCh <- err:
			default: // Never block on an error the consumer is not draining.
			}
			return false
		}

		prefix := s.prefix()
		// Read the current revision first so the watch starts exactly there:
		// a watch registered without one starts at whatever revision the server
		// happens to be on, and would drop changes made in between.
		getOptions := []clientv3.OpOption{clientv3.WithPrefix()}
		if !snapshot {
			getOptions = append(getOptions, clientv3.WithCountOnly())
		}
		resp, err := s.client.KV.Get(ctx, prefix, getOptions...)
		if err != nil {
			fail(errors.Join(exceptions.ExternalFailure, err))
			return
		}
		options := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithRev(resp.Header.Revision + 1),
		}
		if snapshot {
			for _, kv := range resp.Kvs {
				var model M
				if err := json.Unmarshal(kv.Value, &model); err != nil {
					slog.Error("skipping unreadable resource in snapshot", "key", kv.Key, "error", err)
					continue
				}
				model.SetMetadataResourceVersion(strconv.FormatInt(kv.ModRevision, 10))
				if !emit(Event[M]{Type: EventTypePut, Name: s.nameOf(string(kv.Key)), Object: model}) {
					return
				}
			}
			// Terminates the snapshot: everything after it is a change. A
			// replay of puts alone can never tell an informer what disappeared.
			if !emit(Event[M]{Type: EventTypeSync}) {
				return
			}
		}

		watchChan := s.client.Watch(ctx, prefix, options...)
		for watchResp := range watchChan {
			if watchResp.Err() != nil {
				fail(errors.Join(exceptions.ExternalFailure, watchResp.Err()))
				return
			}
			for _, event := range watchResp.Events {
				name := s.nameOf(string(event.Kv.Key))
				if event.Type == clientv3.EventTypeDelete {
					if !emit(Event[M]{Type: EventTypeDelete, Name: name}) {
						return
					}
					continue
				}
				var model M
				if err := json.Unmarshal(event.Kv.Value, &model); err != nil {
					slog.Error("skipping unreadable resource in watch", "key", event.Kv.Key, "error", err)
					continue
				}
				model.SetMetadataResourceVersion(strconv.FormatInt(event.Kv.ModRevision, 10))
				if !emit(Event[M]{Type: EventTypePut, Name: name, Object: model}) {
					return
				}
			}
		}
	}()
	return ch, errCh
}

// nameOf recovers the resource identity from an etcd key under this store.
func (s *store[M]) nameOf(key string) string {
	return strings.TrimPrefix(key, s.prefix())
}

func NewStore[M StorableResource](cli *clientv3.Client, path []string) *store[M] {
	return &store[M]{client: cli, path: path}
}
