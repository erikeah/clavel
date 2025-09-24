package projectv1

import (
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
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

func (self *Project) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *project.Project {
	if self == nil {
		return nil
	}
	conversion := &project.Project{}
	conversion.Name = self.Name
	metadataFmc := fmc.GoTo("metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFmc)
	conversion.Spec = self.Spec.Convert()
	return conversion
}
