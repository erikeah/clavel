package project

import (
	"errors"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/exceptions"
)

func MergeProjectSpecification(over, from *ProjectSpecification) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if over.Flakeref != from.Flakeref {
		over.Flakeref = from.Flakeref
		hasChanged = true
	}
	return hasChanged, nil
}

func MergeProject(over, from *Project) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if from.Name != "" {
		if from.Name != over.Name && over.Name != "" {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("name cannot be changed"))
		}
		over.Name = from.Name
		hasChanged = true
	}

	if specHasChanged, err := MergeProjectSpecification(over.Spec, from.Spec); err != nil {
		return hasChanged, err
	} else if specHasChanged {
		defer over.IncreaseMetadataGeneration()
		hasChanged = true
	}
	if metaHasChanged, err := core.MergeMetadata(over.Metadata, from.Metadata); err != nil {
		return hasChanged, err
	} else if metaHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}
