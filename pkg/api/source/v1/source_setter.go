package sourcev1

import (
	"github.com/erikeah/clavel/internal/source"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
)

func (self *SourceSpecification) Set(spec *source.SourceSpecification) {
	if spec == nil {
		return
	}
	self.Reference = spec.Reference
}

func (self *Source) Set(p *source.Source) {
	if p == nil {
		return
	}
	self.Name = p.Name
	if self.Spec == nil {
		self.Spec = &SourceSpecification{}
	}
	self.Spec.Set(p.Spec)
	if self.Metadata == nil {
		self.Metadata = &corev1.Metadata{}
	}
	self.Metadata.Set(p.Metadata)
}
