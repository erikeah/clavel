package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/erikeah/clavel/internal/exception"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type repository[M any] struct {
	client *clientv3.Client
	path   []string
}

func (r *repository[M]) genPath(name string) string {
	keyPath := append(r.path, name)
	return "/" + strings.Join(keyPath, "/")
}

func (r *repository[M]) FindOne(ctx context.Context, name string) (*M, error) {
	kv := r.client.KV
	resp, err := kv.Get(ctx, r.genPath(name))
	if err != nil {
		return nil, errors.Join(exception.ExternalFailure, err)
	}
	if len(resp.Kvs) < 1 {
		return nil, exception.DoesNotExist
	}
	var model M
	if err := json.Unmarshal(resp.Kvs[0].Value, &model); err != nil {
		return nil, errors.Join(exception.Unknown, err)
	}
	return &model, nil
}

func (r *repository[M]) Add(ctx context.Context, name string, data *M) error {
	kv := r.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Join(exception.Unknown, err)
	}
	destination := r.genPath(name)
	resp, err := kv.
		Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(destination), "=", 0)).
		Then(clientv3.OpPut(destination, string(jsonData))).
		Commit()
	if err != nil {
		return errors.Join(exception.Unknown, err)
	}
	if !resp.Succeeded {
		return errors.Join(exception.AlreadyExist, err)
	}
	return nil
}

func NewRepository[M any](path []string) *repository[M] {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	return &repository[M]{client: cli, path: path}
}
