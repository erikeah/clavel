package core

import (
	"bytes"
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

type CustomResourceDefinition struct {
	Group        string
	Version      string
	Kind         string
	Plural       string
	Schema       []byte
	SpecMessage  string
	ActionModule string
	Metadata     *Metadata
}

func (p CustomResourceDefinition) GetMetadataResourceVersion() string {
	if p.Metadata == nil {
		return ""
	}
	return p.Metadata.GetResourceVersion()
}

func (p CustomResourceDefinition) SetMetadataResourceVersion(rv string) {
	if p.Metadata == nil {
		p.Metadata = &Metadata{}
	}
	p.Metadata.SetResourceVersion(rv)
}

func (p CustomResourceDefinition) IncreaseMetadataGeneration() {
	if p.Metadata == nil {
		p.Metadata = &Metadata{}
	}
	p.Metadata.IncreaseGeneration()
}

func SetDefaults_CustomResourceDefinition(crd *CustomResourceDefinition) error {
	if crd == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if crd.Metadata == nil {
		crd.Metadata = &Metadata{}
	}
	if err := SetDefaults_Metadata(crd.Metadata); err != nil {
		return err
	}
	return nil
}

func MergeCustomResourceDefinition(over, from *CustomResourceDefinition) (bool, error) {
	var hasChanged bool
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if from.Group != "" && over.Group != from.Group {
		over.Group = from.Group
		hasChanged = true
	}
	if from.Version != "" && over.Version != from.Version {
		over.Version = from.Version
		hasChanged = true
	}
	if from.Kind != "" && over.Kind != from.Kind {
		over.Kind = from.Kind
		hasChanged = true
	}
	if from.Plural != "" && over.Plural != from.Plural {
		over.Plural = from.Plural
		hasChanged = true
	}
	if from.Schema != nil && !bytes.Equal(over.Schema, from.Schema) {
		over.Schema = from.Schema
		hasChanged = true
	}
	if from.SpecMessage != "" && over.SpecMessage != from.SpecMessage {
		over.SpecMessage = from.SpecMessage
		hasChanged = true
	}
	if from.ActionModule != "" && over.ActionModule != from.ActionModule {
		over.ActionModule = from.ActionModule
		hasChanged = true
	}
	if metaHasChanged, err := MergeMetadata(over.Metadata, from.Metadata); err != nil {
		return hasChanged, err
	} else if metaHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}

func ValidateCustomResourceDefinition(crd *CustomResourceDefinition) error {
	var crdErrors error
	err := ValidateName(crd.Group)
	crdErrors = errors.Join(crdErrors, groupVersionName("invalid group", err))
	err = ValidateName(crd.Version)
	crdErrors = errors.Join(crdErrors, groupVersionName("invalid version", err))
	err = ValidateName(crd.Kind)
	crdErrors = errors.Join(crdErrors, groupVersionName("invalid kind", err))
	err = ValidateName(crd.Plural)
	crdErrors = errors.Join(crdErrors, groupVersionName("invalid plural", err))
	if len(crd.Schema) == 0 {
		crdErrors = errors.Join(crdErrors, errors.New("Schema cannot be empty"))
	}
	if crd.SpecMessage == "" {
		crdErrors = errors.Join(crdErrors, errors.New("SpecMessage cannot be empty"))
	}
	return crdErrors
}
