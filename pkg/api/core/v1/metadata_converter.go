package corev1

import (
	"time"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *Metadata) Convert(fm *fieldmaskcommander.FieldMaskCommander) *core.Metadata {
	if self == nil {
		return nil
	}
	meta := &core.Metadata{}
	var creationTS, deletionTS *time.Time
	if self.CreationTimestamp != "" {
		t, err := time.Parse(time.RFC3339, self.CreationTimestamp)
		if err == nil {
			creationTS = &t
		}
	}
	if self.DeletionTimestamp != "" {
		t, err := time.Parse(time.RFC3339, self.DeletionTimestamp)
		if err == nil {
			deletionTS = &t
		}
	}
	meta.Generation = self.GetGeneration()
	if fm.IsFieldMasked("finalizers") {
		if self.Finalizers == nil {
			meta.Finalizers = []string{}
		} else {
			meta.Finalizers = self.Finalizers
		}
	}
	meta.CreationTimestamp = creationTS
	meta.DeletionTimestamp = deletionTS
	return meta
}
