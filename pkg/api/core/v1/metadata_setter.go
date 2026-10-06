package corev1

import (
	"time"

	"github.com/erikeah/clavel/internal/core"
)

func (self *OwnerReference) Set(reference *core.OwnerReference) {
	if reference == nil {
		return
	}
	self.ApiVersion = reference.APIVersion
	self.Kind = reference.Kind
	self.Name = reference.Name
	self.Uid = reference.UID
	self.Controller = reference.Controller
	self.BlockOwnerDeletion = reference.BlockOwnerDeletion
}

func SetOwnerReferences(references []core.OwnerReference) []*OwnerReference {
	if references == nil {
		return nil
	}
	conversions := make([]*OwnerReference, 0, len(references))
	for i := range references {
		conversion := &OwnerReference{}
		conversion.Set(&references[i])
		conversions = append(conversions, conversion)
	}
	return conversions
}

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
	self.Name = meta.Name
	self.Generation = meta.Generation
	self.ResourceVersion = meta.ResourceVersion
	self.Uid = meta.UID
	self.GenerateName = meta.GenerateName
	self.Namespace = meta.Namespace
	self.CreationTimestamp = creationTS
	self.DeletionTimestamp = deletionTS
	self.Finalizers = meta.Finalizers
	self.Labels = meta.Labels
	self.Annotations = meta.Annotations
	self.OwnerReferences = SetOwnerReferences(meta.OwnerReferences)
}
