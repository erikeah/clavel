package genericstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/erikeah/clavel/internal/exceptions"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type store[M any] struct {
	client *clientv3.Client
	path   []string
}

func (r *store[M]) genPath(name string) string {
	keyPath := append(r.path, name)
	return "/" + strings.Join(keyPath, "/")
}

func (r *store[M]) FindOne(ctx context.Context, name string) (*M, error) {
	kv := r.client.KV
	resp, err := kv.Get(ctx, r.genPath(name))
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
	return &model, nil
}

func (r *store[M]) List(ctx context.Context) ([]*M, error) {
	kv := r.client.KV
	resp, err := kv.Get(ctx, r.genPath(""), clientv3.WithPrefix())
	if err != nil {
		return nil, errors.Join(exceptions.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		return nil, exceptions.DoesNotExist
	}
	var list []*M
	for _, value := range resp.Kvs {
		var model M
		if err := json.Unmarshal(value.Value, &model); err != nil {
			return nil, errors.Join(exceptions.Unknown, err)
		}
		list = append(list, &model)
	}
	return list, nil
}

func (r *store[M]) Create(ctx context.Context, name string, data *M) error {
	kv := r.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Join(exceptions.Unknown, err)
	}
	destination := r.genPath(name)
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

func (r *store[M]) Delete(ctx context.Context, name string) error {
	kv := r.client.KV
	resp, err := kv.Delete(ctx, r.genPath(name))
	if err != nil {
		return errors.Join(exceptions.ExternalFailure, err)
	}
	if resp.Deleted == 0 {
		return exceptions.DoesNotExist
	}
	return nil
}

func NewStore[M any](cli *clientv3.Client, path []string) *store[M] {
	return &store[M]{client: cli, path: path}
}
