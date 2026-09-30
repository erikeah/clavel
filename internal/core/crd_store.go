package core

import (
	"context"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/erikeah/clavel/internal/genericstore"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type CustomResourceDefinitionStore interface {
	Create(ctx context.Context, group string, version string, kind string, data *CustomResourceDefinition) error
	Delete(ctx context.Context, group string, version string, kind string) error
	FindOne(ctx context.Context, group string, version string, kind string) (*CustomResourceDefinition, error)
	FindByPlural(ctx context.Context, group string, version string, plural string) (*CustomResourceDefinition, error)
	List(ctx context.Context) ([]*CustomResourceDefinition, error)
	WatchAll(ctx context.Context) (<-chan *CustomResourceDefinition, <-chan error)
}

type crdStore struct {
	client *clientv3.Client
}

func NewCustomResourceDefinitionStore(cli *clientv3.Client) CustomResourceDefinitionStore {
	return &crdStore{client: cli}
}

func (s *crdStore) Create(ctx context.Context, group string, version string, kind string, data *CustomResourceDefinition) error {
	return genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd", group, version}).Create(ctx, kind, data)
}

func (s *crdStore) Delete(ctx context.Context, group string, version string, kind string) error {
	return genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd", group, version}).Delete(ctx, kind)
}

func (s *crdStore) FindOne(ctx context.Context, group string, version string, kind string) (*CustomResourceDefinition, error) {
	return genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd", group, version}).FindOne(ctx, kind)
}

func (s *crdStore) FindByPlural(ctx context.Context, group string, version string, plural string) (*CustomResourceDefinition, error) {
	crds, err := genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd", group, version}).List(ctx)
	if err != nil {
		return nil, err
	}
	for _, crd := range crds {
		if crd.Plural == plural {
			return crd, nil
		}
	}
	return nil, errors.Join(exceptions.DoesNotExist, errors.New("kind for plural not registered"))
}

func (s *crdStore) List(ctx context.Context) ([]*CustomResourceDefinition, error) {
	return genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd"}).List(ctx)
}

func (s *crdStore) WatchAll(ctx context.Context) (<-chan *CustomResourceDefinition, <-chan error) {
	return genericstore.NewStore[CustomResourceDefinition](s.client, []string{"crd"}).Watch(ctx)
}
