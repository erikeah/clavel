package core

import (
	"bytes"
	"context"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

type CustomResourceDefinitionService struct {
	store       CustomResourceDefinitionStore
	resources   ResourceStoreProvider
	setDefaults func(*CustomResourceDefinition) error
	merge       func(over, from *CustomResourceDefinition) (bool, error)
	validate    func(*CustomResourceDefinition) error
}

func (s *CustomResourceDefinitionService) Create(ctx context.Context, data *CustomResourceDefinition) error {
	resource := &CustomResourceDefinition{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if _, err := s.merge(resource, data); err != nil {
		return err
	}
	if err := s.validate(resource); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	existing, err := s.store.FindOne(ctx, resource.Group, resource.Version, resource.Kind)
	if err == nil {
		if customResourceDefinitionsEqual(existing, resource) {
			// Idempotent apply: identical definition already registered.
			return nil
		}
		return errors.Join(exceptions.AlreadyExist, errors.New("CustomResourceDefinition already registered with a different schema"))
	}
	if !errors.Is(err, exceptions.DoesNotExist) {
		return err
	}
	return s.store.Create(ctx, resource.Group, resource.Version, resource.Kind, resource)
}

func (s *CustomResourceDefinitionService) Delete(ctx context.Context, group string, version string, kind string) error {
	crd, err := s.store.FindOne(ctx, group, version, kind)
	if err != nil {
		return err
	}
	resources, err := s.resources(group, version, crd.Plural).List(ctx)
	if err != nil {
		return err
	}
	if len(resources) > 0 {
		return errors.Join(exceptions.InvalidArguments, errors.New("cannot delete CustomResourceDefinition while resources exist"))
	}
	return s.store.Delete(ctx, group, version, kind)
}

func (s *CustomResourceDefinitionService) Show(ctx context.Context, group string, version string, kind string) (*CustomResourceDefinition, error) {
	return s.store.FindOne(ctx, group, version, kind)
}

func (s *CustomResourceDefinitionService) List(ctx context.Context) ([]*CustomResourceDefinition, error) {
	return s.store.List(ctx)
}

func (s *CustomResourceDefinitionService) Watch(ctx context.Context) (<-chan *CustomResourceDefinition, <-chan error) {
	return s.store.WatchAll(ctx)
}

func customResourceDefinitionsEqual(a, b *CustomResourceDefinition) bool {
	return a.Group == b.Group &&
		a.Version == b.Version &&
		a.Kind == b.Kind &&
		a.Plural == b.Plural &&
		bytes.Equal(a.Schema, b.Schema) &&
		a.SpecMessage == b.SpecMessage &&
		a.ActionModule == b.ActionModule
}

func NewCustomResourceDefinitionService(
	store CustomResourceDefinitionStore,
	resources ResourceStoreProvider,
) *CustomResourceDefinitionService {
	return &CustomResourceDefinitionService{
		store:       store,
		resources:   resources,
		setDefaults: SetDefaults_CustomResourceDefinition,
		merge:       MergeCustomResourceDefinition,
		validate:    ValidateCustomResourceDefinition,
	}
}
