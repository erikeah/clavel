package core

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

type CustomResourceService struct {
	crds         CustomResourceDefinitionStore
	resources    ResourceStoreProvider
	setDefaults  func(*Resource) error
	merge        func(over, from *Resource) (bool, error)
	validate     func(*Resource) error
	validateSpec func(schema []byte, specMessage string, spec json.RawMessage) (json.RawMessage, error)
}

func (s *CustomResourceService) resolveCRD(ctx context.Context, group string, version string, plural string) (*CustomResourceDefinition, error) {
	crd, err := s.crds.FindByPlural(ctx, group, version, plural)
	if err != nil {
		return nil, errors.Join(exceptions.DoesNotExist, errors.New("CustomResourceDefinition not registered"))
	}
	return crd, nil
}

func (s *CustomResourceService) Create(ctx context.Context, data *Resource) error {
	crd, err := s.resolveCRD(ctx, data.Group, data.Version, data.Plural)
	if err != nil {
		return err
	}
	resource := &Resource{}
	if err := s.setDefaults(resource); err != nil {
		return err
	}
	if _, err := s.merge(resource, data); err != nil {
		return err
	}
	resource.Group = data.Group
	resource.Version = data.Version
	resource.Plural = data.Plural
	resource.Kind = crd.Kind
	if err := s.validate(resource); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	canonical, err := s.validateSpec(crd.Schema, crd.SpecMessage, resource.Spec)
	if err != nil {
		return err
	}
	resource.Spec = canonical
	return s.resources(data.Group, data.Version, data.Plural).Create(ctx, resource.Name, resource)
}

func (s *CustomResourceService) Update(ctx context.Context, group string, version string, plural string, name string, data *Resource) error {
	crd, err := s.resolveCRD(ctx, group, version, plural)
	if err != nil {
		return err
	}
	target, err := s.resources(group, version, plural).FindOne(ctx, name)
	if err != nil {
		return err
	}
	if hasChanged, err := s.merge(target, data); err != nil {
		return errors.Join(exceptions.InternalFailure, err)
	} else if !hasChanged {
		return exceptions.NotModified
	}
	target.Group = group
	target.Version = version
	target.Plural = plural
	target.Kind = crd.Kind
	if err := s.validate(target); err != nil {
		return errors.Join(exceptions.InvalidArguments, err)
	}
	canonical, err := s.validateSpec(crd.Schema, crd.SpecMessage, target.Spec)
	if err != nil {
		return err
	}
	target.Spec = canonical
	return s.resources(group, version, plural).Update(ctx, target.Name, target)
}

func (s *CustomResourceService) Delete(ctx context.Context, group string, version string, plural string, name string) error {
	if _, err := s.resolveCRD(ctx, group, version, plural); err != nil {
		return err
	}
	return s.resources(group, version, plural).Delete(ctx, name)
}

func (s *CustomResourceService) List(ctx context.Context, group string, version string, plural string) ([]*Resource, error) {
	if _, err := s.resolveCRD(ctx, group, version, plural); err != nil {
		return nil, err
	}
	return s.resources(group, version, plural).List(ctx)
}

func (s *CustomResourceService) Show(ctx context.Context, group string, version string, plural string, name string) (*Resource, error) {
	if _, err := s.resolveCRD(ctx, group, version, plural); err != nil {
		return nil, err
	}
	return s.resources(group, version, plural).FindOne(ctx, name)
}

func (s *CustomResourceService) Watch(ctx context.Context, group string, version string, plural string) (<-chan *Resource, <-chan error) {
	ch := make(chan *Resource)
	errCh := make(chan error, 1)
	crd, err := s.resolveCRD(ctx, group, version, plural)
	if err != nil {
		errCh <- err
		close(ch)
		return ch, errCh
	}
	resourceChan, resourceErrChan := s.resources(group, version, plural).Watch(ctx)
	go func() {
		defer close(ch)
		defer close(errCh)
		for {
			select {
			case resource, ok := <-resourceChan:
				if !ok {
					return
				}
				resource.Kind = crd.Kind
				ch <- resource
			case err := <-resourceErrChan:
				if err != nil {
					errCh <- err
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, errCh
}

func NewCustomResourceService(
	crds CustomResourceDefinitionStore,
	resources ResourceStoreProvider,
) *CustomResourceService {
	return &CustomResourceService{
		crds:         crds,
		resources:    resources,
		setDefaults:  SetDefaults_Resource,
		merge:        MergeResource,
		validate:     ValidateResource,
		validateSpec: ValidateAndCanonicalizeSpec,
	}
}
