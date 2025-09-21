package projectv1

import (
	"github.com/erikeah/clavel/internal/project"
	corev1 "github.com/erikeah/clavel/pkg/api/core/v1"
)

func (self *ProjectSpecification) Set(spec *project.ProjectSpecification) {
	if spec == nil {
		return
	}
	self.Flakeref = spec.Flakeref
}

func (self *Project) Set(p *project.Project) {
	if p == nil {
		return
	}
	self.Name = p.Name
	if self.Spec == nil {
		self.Spec = &ProjectSpecification{}
	}
	self.Spec.Set(p.Spec)
	if self.Metadata == nil {
		self.Metadata = &corev1.Metadata{}
	}
	self.Metadata.Set(p.Metadata)
}
