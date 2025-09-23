package project

import (
	"errors"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/exceptions"
)

func MergeProjectSpecification(over, from *ProjectSpecification) error {
	if over == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if from != nil {
		over.Flakeref = from.Flakeref
	}
	return nil
}

func MergeProject(over, from *Project) error {
	if over == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if from.Name != "" {
		if from.Name != over.Name && over.Name != "" {
			return errors.Join(exceptions.InvalidArguments, errors.New("name cannot be changed"))
		}
		over.Name = from.Name
	}
	if err := MergeProjectSpecification(over.Spec, from.Spec); err != nil {
		return err
	}
	if err := core.MergeMetadata(over.Metadata, from.Metadata); err != nil {
		return err
	}
	return nil
}
