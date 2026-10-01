package corev1

import (
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *EvaluationSpecification) Convert() core.EvaluationSpecification {
	if self == nil {
		return core.EvaluationSpecification{}
	}
	return core.EvaluationSpecification{
		Reference: self.Reference,
	}
}

func (self *EvaluationStatusPhase) Convert() core.EvaluationPhase {
	if self == nil {
		return core.EvaluationPhaseUnspecified
	}
	return core.EvaluationPhase(*self)
}

func (self *EvaluationStatus) Convert() core.EvaluationStatus {
	if self == nil {
		return core.EvaluationStatus{}
	}
	return core.EvaluationStatus{
		Phase:     self.Phase.Convert(),
		StorePath: self.StorePath,
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
	conversion.Status = self.Status.Convert()
	return conversion
}
