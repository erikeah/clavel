package corev1

import (
	"time"

	"github.com/erikeah/clavel/internal/core"
)

func (self *Metadata) Set(meta *core.Metadata) {
	if meta == nil {
		return
	}
	var creationTS, deletionTS string
	if meta.CreationTimestamp != nil {
		creationTS = meta.CreationTimestamp.Format(time.RFC3339)
	}
	if meta.DeletionTimestamp != nil {
		deletionTS = meta.DeletionTimestamp.Format(time.RFC3339)
	}
	self.Generation = meta.Generation
	self.CreationTimestamp = creationTS
	self.DeletionTimestamp = deletionTS
	self.Finalizers = meta.Finalizers
}
