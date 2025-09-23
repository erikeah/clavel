package projectv1

import (
	"github.com/erikeah/clavel/internal/project"
	"github.com/erikeah/clavel/internal/utils"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (self *ProjectSpecification) Convert() *project.ProjectSpecification {
	if self == nil {
		return nil
	}
	return &project.ProjectSpecification{
		Flakeref: self.Flakeref,
	}
}

func (self *Project) Convert(fm *fieldmaskpb.FieldMask) *project.Project {
	if self == nil {
		return nil
	}
	conversion := &project.Project{}
	conversion.Name = self.Name
	metadataFm, _ := utils.RelocateFieldMask(self.Metadata, fm, "metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFm)
	conversion.Spec = self.Spec.Convert()
	return conversion
}
