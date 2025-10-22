package sourcev1

import (
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
	"github.com/erikeah/clavel/internal/source"
)

func (self *SourceSpecification) Convert() *source.SourceSpecification {
	if self == nil {
		return nil
	}
	return &source.SourceSpecification{
		Reference: self.Reference,
	}
}

func (self *Source) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *source.Source {
	if self == nil {
		return nil
	}
	conversion := &source.Source{}
	conversion.Name = self.Name
	metadataFmc := fmc.GoTo("metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFmc)
	conversion.Spec = self.Spec.Convert()
	return conversion
}
