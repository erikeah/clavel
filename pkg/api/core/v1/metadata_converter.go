package corev1

import (
	"time"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *OwnerReference) Convert() core.OwnerReference {
	if self == nil {
		return core.OwnerReference{}
	}
	return core.OwnerReference{
		APIVersion:         self.GetApiVersion(),
		Kind:               self.GetKind(),
		Name:               self.GetName(),
		UID:                self.GetUid(),
		Controller:         self.GetController(),
		BlockOwnerDeletion: self.GetBlockOwnerDeletion(),
	}
}

func ConvertOwnerReferences(references []*OwnerReference) []core.OwnerReference {
	if references == nil {
		return nil
	}
	conversions := make([]core.OwnerReference, 0, len(references))
	for _, reference := range references {
		conversions = append(conversions, reference.Convert())
	}
	return conversions
}

func (self *Metadata) Convert(fmc *fieldmaskcommander.FieldMaskCommander) core.Metadata {
	if self == nil {
		return core.Metadata{}
	}
	meta := core.Metadata{}
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
	// HINT: Name is the request identity, it is carried through unmasked.
	meta.Name = self.GetName()
	meta.Generation = self.GetGeneration()
	meta.ResourceVersion = self.GetResourceVersion()
	meta.UID = self.GetUid()
	meta.GenerateName = self.GetGenerateName()
	meta.Namespace = self.GetNamespace()
	if fmc.IsFieldMasked("finalizers") {
		if self.Finalizers == nil {
			meta.Finalizers = []string{}
		} else {
			meta.Finalizers = self.Finalizers
		}
	}
	if fmc.IsFieldMasked("labels") {
		if self.Labels == nil {
			meta.Labels = map[string]string{}
		} else {
			meta.Labels = self.Labels
		}
	}
	if fmc.IsFieldMasked("annotations") {
		if self.Annotations == nil {
			meta.Annotations = map[string]string{}
		} else {
			meta.Annotations = self.Annotations
		}
	}
	if fmc.IsFieldMasked("ownerReferences") {
		if self.OwnerReferences == nil {
			meta.OwnerReferences = []core.OwnerReference{}
		} else {
			meta.OwnerReferences = ConvertOwnerReferences(self.OwnerReferences)
		}
	}
	meta.CreationTimestamp = creationTS
	meta.DeletionTimestamp = deletionTS
	return meta
}
