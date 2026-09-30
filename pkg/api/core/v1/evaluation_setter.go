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

func (self *Evaluation) Set(p *core.Evaluation) {
	if p == nil {
		return
	}
	self.Name = p.Name
	if self.Spec == nil {
		self.Spec = &EvaluationSpecification{}
	}
	self.Spec.Set(p.Spec)
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(p.Metadata)
}
