package core

import (
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

func MergeMetadata(over, from *Metadata) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InvalidArguments
	}
	// HINT: Perform optimist concurrency validation
	if over.ResourceVersion != from.ResourceVersion {
		return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("resourceVersion does not match"))
	}
	if from.Finalizers != nil {
		over.Finalizers = from.Finalizers
		hasChanged = true
	}
	if from.DeletionTimestamp != nil && (over.DeletionTimestamp == nil || from.DeletionTimestamp.After(*over.DeletionTimestamp)) {
		over.DeletionTimestamp = from.DeletionTimestamp
		hasChanged = true
	}
	// Generation and CreationTimestamp is not merged on purpose
	return hasChanged, nil
}

func MergeEvaluationSpecification(over, from *EvaluationSpecification) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if over.Reference != from.Reference {
		over.Reference = from.Reference
		hasChanged = true
	}
	return hasChanged, nil
}

func MergeEvaluation(over, from *Evaluation) (bool, error) {
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
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("Name cannot be changed"))
		}
		over.Name = from.Name
		hasChanged = true
	}
	if specHasChanged, err := MergeEvaluationSpecification(over.Spec, from.Spec); err != nil {
		return hasChanged, err
	} else if specHasChanged {
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
