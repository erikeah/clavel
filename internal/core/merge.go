package core

import (
	"errors"

	"github.com/erikeah/clavel/internal/exceptions"
)

func MergeMetadata(over, from *Metadata) error {
	if over == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if over.Generation != from.Generation {
		return errors.Join(exceptions.InvalidArguments, errors.New("generation does not match"))
	}
	if from.Finalizers != nil {
		over.Finalizers = from.Finalizers
	}
	if from.DeletionTimestamp != nil {
		over.DeletionTimestamp = from.DeletionTimestamp
	}
	// CreationTimestamp is not merged on purpose
	return nil
}
