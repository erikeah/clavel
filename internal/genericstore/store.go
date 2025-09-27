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

type store[M StorableResource] struct {
	client *clientv3.Client
	path   []string
}

func (s *store[M]) genPath(name string) string {
	keyPath := append(s.path, name)
	return "/" + strings.Join(keyPath, "/")
}

func (s *store[M]) FindOne(ctx context.Context, name string) (*M, error) {
	kv := s.client.KV
	resp, err := kv.Get(ctx, s.genPath(name))
	if err != nil {
		return nil, errors.Join(exceptions.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		return nil, exceptions.DoesNotExist
	}
	var model M
	if err := json.Unmarshal(resp.Kvs[0].Value, &model); err != nil {
		return nil, errors.Join(exceptions.Unknown, err)
	}
	model.SetMetadataResourceVersion(strconv.FormatInt(resp.Kvs[0].ModRevision, 10))
	return &model, nil
}

func (s *store[M]) List(ctx context.Context) ([]*M, error) {
	kv := s.client.KV
	resp, err := kv.Get(ctx, s.genPath(""), clientv3.WithPrefix())
	if err != nil {
		return nil, errors.Join(exceptions.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		return nil, exceptions.DoesNotExist
	}
	var list []*M
	for i, value := range resp.Kvs {
		var model M
		if err := json.Unmarshal(value.Value, &model); err != nil {
			return nil, errors.Join(exceptions.Unknown, err)
		}
		model.SetMetadataResourceVersion(strconv.FormatInt(resp.Kvs[i].ModRevision, 10))
		list = append(list, &model)
	}
	return list, nil
}

func (s *store[M]) Create(ctx context.Context, name string, data *M) error {
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

func (s *store[M]) Update(ctx context.Context, key string, data *M) error {
	kv := s.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	destination := s.genPath(key)
	_, err = kv.Put(ctx, destination, string(jsonData))
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	return nil
}

func (s *store[M]) Watch(ctx context.Context) (<-chan *M, <-chan error) {
	ch := make(chan *M)
	errCh := make(chan error, 1) // Buffered channel for errors

	go func() {
		defer close(ch)
		defer close(errCh)
		watchChan := s.client.Watch(ctx, s.genPath(""), clientv3.WithPrefix())
		for watchResp := range watchChan {
			if watchResp.Err() != nil {
				errCh <- errors.Join(exceptions.ExternalFailure, watchResp.Err())
				return
			}
			for _, event := range watchResp.Events {
				var model M
				if event.Type == clientv3.EventTypePut {
					if err := json.Unmarshal(event.Kv.Value, &model); err != nil {
						errCh <- errors.Join(exceptions.Unknown, err)
						slog.Error(err.Error())
					}
					ch <- &model
				}
			}
		}
	}()
	return ch, errCh
}

func NewStore[M StorableResource](cli *clientv3.Client, path []string) *store[M] {
	return &store[M]{client: cli, path: path}
}
