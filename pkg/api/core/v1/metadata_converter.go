package corev1

import (
	"time"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/utils"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (self *Metadata) Convert(fm *fieldmaskpb.FieldMask) *core.Metadata {
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
	if utils.IsFieldMasked(fm, "finalizers") {
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
