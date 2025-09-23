package core

import (
	"errors"
)

func ValidateMetadata(m *Metadata) error {
	var metadataErrors error
	if m.CreationTimestamp == nil {
		metadataErrors = errors.Join(metadataErrors, errors.New("CreationTimestamp cannot be empty"))
	}
	return metadataErrors
}
