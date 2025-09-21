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

func (self *Metadata) Convert() *core.Metadata {
	if self == nil {
		return nil
	}
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
	return &core.Metadata{
		Generation:        self.Generation,
		Finalizers:        self.Finalizers,
		CreationTimestamp: creationTS,
		DeletionTimestamp: deletionTS,
	}
}
