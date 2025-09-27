package core

import (
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

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
