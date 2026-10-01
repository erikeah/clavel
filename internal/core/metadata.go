package core

import (
	"errors"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

type Metadata struct {
	generationHasIncrease bool
	Generation            int64      `json:"generation"`
	ResourceVersion       string     `json:"-"`
	Finalizers            []string   `json:"finalizers,omitempty"`
	CreationTimestamp     *time.Time `json:"creationTimestamp"`
	DeletionTimestamp     *time.Time `json:"deletionTimestamp,omitempty"`
}

func (m *Metadata) IncreaseGeneration() {
	if m.generationHasIncrease {
		return
	}
	m.Generation++
	m.generationHasIncrease = true
}

func (m *Metadata) SetResourceVersion(resourceVersion string) {
	m.ResourceVersion = resourceVersion
}

func (m *Metadata) GetResourceVersion() string {
	if m == nil {
		return ""
	}
	return m.ResourceVersion
}

func SetDefaults_Metadata(m *Metadata) error {
	if m == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if m.CreationTimestamp == nil {
		now := time.Now().UTC()
		m.CreationTimestamp = &now
		// HINT: Following has been set because the resource is new
		m.Generation = -1
		m.ResourceVersion = ""
	}
	return nil
}

func ValidateMetadata(m *Metadata) error {
	var metadataErrors error
	if m == nil {
		metadataErrors = errors.Join(metadataErrors, errors.New("Metadata cannot be nil"))
		return metadataErrors
	}
	if m.CreationTimestamp == nil {
		metadataErrors = errors.Join(metadataErrors, errors.New("CreationTimestamp cannot be empty"))
	}
	return metadataErrors
}

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
