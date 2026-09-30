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

func (self *EvaluationStatusResult) Convert() *core.EvaluationStatusResult {
	if self == nil {
		return nil
	}
	return &core.EvaluationStatusResult{
		Hash:      self.Hash,
		StorePath: self.StorePath,
	}
}

func (self *EvaluationStatus) Convert() *core.EvaluationStatus {
	if self == nil {
		return nil
	}
	status := &core.EvaluationStatus{
		Result: self.Result.Convert(),
	}
	switch self.GetPhase() {
	case EvaluationStatusPhase_EVALUATION_STATUS_PHASE_READY:
		status.Phase = core.EvaluationStatusPhaseReady
	case EvaluationStatusPhase_EVALUATION_STATUS_PHASE_FAILED:
		status.Phase = core.EvaluationStatusPhaseFailed
	}
	return status
}

func (self *Evaluation) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *core.Evaluation {
	if self == nil {
		return nil
	}
	conversion := &core.Evaluation{}
	conversion.Name = self.Name
	metadataFmc := fmc.GoTo("metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFmc)
	conversion.Status = self.Status.Convert()
	conversion.Spec = self.Spec.Convert()
	return conversion
}
