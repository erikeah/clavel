package corev1

import (
	"github.com/erikeah/clavel/internal/core"
)

func (self *EvaluationSpecification) Set(spec *core.EvaluationSpecification) {
	if spec == nil {
		return
	}
	self.Reference = spec.Reference
}

func (self *EvaluationStatusResult) Set(result *core.EvaluationStatusResult) {
	if result == nil {
		return
	}
	self.Hash = result.Hash
	self.StorePath = result.StorePath
}

func (self *EvaluationStatus) Set(status *core.EvaluationStatus) {
	if status == nil {
		return
	}
	switch status.Phase {
	case core.EvaluationStatusPhaseReady:
		self.Phase = EvaluationStatusPhase_EVALUATION_STATUS_PHASE_READY
	case core.EvaluationStatusPhaseFailed:
		self.Phase = EvaluationStatusPhase_EVALUATION_STATUS_PHASE_FAILED
	default:
		self.Phase = EvaluationStatusPhase_EVALUATION_STATUS_PHASE_UNSPECIFIED
	}
	if self.Result == nil {
		self.Result = &EvaluationStatusResult{}
	}
	self.Result.Set(status.Result)
}

func (self *Evaluation) Set(p *core.Evaluation) {
	if p == nil {
		return
	}
	self.Name = p.Name
	if self.Spec == nil {
		self.Spec = &EvaluationSpecification{}
	}
	self.Spec.Set(p.Spec)
	if self.Status == nil {
		self.Status = &EvaluationStatus{}
	}
	self.Status.Set(p.Status)
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(p.Metadata)
}
