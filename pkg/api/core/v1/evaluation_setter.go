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

func (self *EvaluationStatus) Set(status *core.EvaluationStatus) {
	if status == nil {
		return
	}
	self.Phase = EvaluationStatusPhase(status.Phase)
	self.Result = status.Result
	self.ObservedGeneration = status.ObservedGeneration
	self.Message = status.Message
}

func (self *Evaluation) Set(p *core.Evaluation) {
	if p == nil {
		return
	}
	self.ApiVersion = p.APIVersion
	self.Kind = p.Kind
	if self.Spec == nil {
		self.Spec = &EvaluationSpecification{}
	}
	self.Spec.Set(&p.Spec)
	if self.Status == nil {
		self.Status = &EvaluationStatus{}
	}
	self.Status.Set(&p.Status)
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(&p.Metadata)
}
