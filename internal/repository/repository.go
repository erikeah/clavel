package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/erikeah/clavel/internal/errors"
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

func (r *repository[M]) FindOne(ctx context.Context, name string) (*M, errors.Error) {
	kv := r.client.KV
	resp, err := kv.Get(ctx, r.genPath(name))
	if err != nil {
		return nil, errors.New(errors.Unknown, err.Error())
	}
	if len(resp.Kvs) < 1 {
		return nil, errors.New(errors.DoesNotExist, fmt.Sprintf("Resource %s has not been found.", name))
	}
	var model M
	if err := json.Unmarshal(resp.Kvs[0].Value, &model); err != nil {
		return nil, errors.New(errors.Unknown, err.Error())
	}
	return &model, nil
}

func (r *repository[M]) Add(ctx context.Context, name string, data *M) errors.Error {
	kv := r.client.KV
	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.New(errors.Unknown, err.Error())
	}
	destination := r.genPath(name)
	resp, err := kv.
		Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(destination), "=", 0)).
		Then(clientv3.OpPut(destination, string(jsonData))).
		Commit()
	if err != nil {
		return errors.New(errors.Unknown, err.Error())
	}
	if !resp.Succeeded {
		return errors.New(errors.ResourceAlreadyExist, fmt.Sprintf("Resource %s already exist", name))
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
