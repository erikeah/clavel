package projectv1

import (
	"github.com/erikeah/clavel/internal/project"
)

func (self *ProjectSpecification) Convert() *project.ProjectSpecification {
	if self == nil {
		return nil
	}
	return &project.ProjectSpecification{
		Flakeref: self.Flakeref,
	}
}

func (self *Project) Convert() *project.Project {
	if self == nil {
		return nil
	}
	conversion := &project.Project{}
	conversion.Name = self.Name
	conversion.Metadata = self.Metadata.Convert()
	conversion.Spec = self.Spec.Convert()
	return conversion
}
