package core

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

type Resource struct {
	Group    string
	Version  string
	Plural   string
	Kind     string
	Name     string
	Metadata *Metadata
	Spec     json.RawMessage
}

func (p Resource) GetMetadataResourceVersion() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata.GetResourceVersion()
}

func (p Resource) SetMetadataResourceVersion(rv string) {
	if p.Metadata == nil {
		p.Metadata = &Metadata{}
	}
	p.Metadata.SetResourceVersion(rv)
}

func (p Resource) IncreaseMetadataGeneration() {
	if p.Metadata == nil {
		p.Metadata = &Metadata{}
	}
	p.Metadata.IncreaseGeneration()
}

func SetDefaults_Resource(resource *Resource) error {
	if resource == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if resource.Metadata == nil {
		resource.Metadata = &Metadata{}
	}
	if err := SetDefaults_Metadata(resource.Metadata); err != nil {
		return err
	}
	return nil
}

func MergeResource(over, from *Resource) (bool, error) {
	var hasChanged bool
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if from.Name != "" {
		if from.Name != over.Name && over.Name != "" {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("Name cannot be changed"))
		}
		over.Name = from.Name
		hasChanged = true
	}
	if from.Group != "" && over.Group != from.Group {
		over.Group = from.Group
		hasChanged = true
	}
	if from.Version != "" && over.Version != from.Version {
		over.Version = from.Version
		hasChanged = true
	}
	if from.Plural != "" && over.Plural != from.Plural {
		over.Plural = from.Plural
		hasChanged = true
	}
	if from.Kind != "" && over.Kind != from.Kind {
		over.Kind = from.Kind
		hasChanged = true
	}
	if from.Spec != nil && !bytes.Equal(over.Spec, from.Spec) {
		over.Spec = from.Spec
		defer over.IncreaseMetadataGeneration()
		hasChanged = true
	}
	if metaHasChanged, err := MergeMetadata(over.Metadata, from.Metadata); err != nil {
		return hasChanged, err
	} else if metaHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}

func ValidateResource(resource *Resource) error {
	var resourceErrors error
	err := ValidateName(resource.Name)
	resourceErrors = errors.Join(resourceErrors, err)
	err = ValidateName(resource.Group)
	resourceErrors = errors.Join(resourceErrors, groupVersionName("invalid group", err))
	err = ValidateName(resource.Version)
	resourceErrors = errors.Join(resourceErrors, groupVersionName("invalid version", err))
	err = ValidateName(resource.Plural)
	resourceErrors = errors.Join(resourceErrors, groupVersionName("invalid plural", err))
	if len(resource.Spec) == 0 {
		resourceErrors = errors.Join(resourceErrors, errors.New("Spec cannot be empty"))
	}
	if err := ValidateMetadata(resource.Metadata); err != nil {
		resourceErrors = errors.Join(resourceErrors, err)
	}
	return resourceErrors
}
