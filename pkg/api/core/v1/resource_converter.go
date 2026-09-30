package corev1

import (
	"encoding/json"

	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *Resource) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *core.Resource {
	if self == nil {
		return nil
	}
	conversion := &core.Resource{}
	conversion.Name = self.Name
	conversion.Group = self.Group
	conversion.Version = self.Version
	conversion.Plural = self.Plural
	conversion.Kind = self.Kind
	conversion.Metadata = self.Metadata.Convert(fmc.GoTo("metadata"))
	conversion.Spec = json.RawMessage(self.Spec)
	return conversion
}
