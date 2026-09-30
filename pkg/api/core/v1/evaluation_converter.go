package corev1

import (
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *EvaluationSpecification) Convert() *core.EvaluationSpecification {
	if self == nil {
		return nil
	}
	return &core.EvaluationSpecification{
		Reference: self.Reference,
	}
}

func (self *Evaluation) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *core.Evaluation {
	if self == nil {
		return nil
	}
	conversion := &core.Evaluation{}
	conversion.Name = self.Name
	metadataFmc := fmc.GoTo("metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFmc)
	conversion.Spec = self.Spec.Convert()
	return conversion
}
