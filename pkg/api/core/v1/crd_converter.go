package corev1

import (
	"github.com/erikeah/clavel/internal/core"
)

func (self *CustomResourceDefinition) Convert() *core.CustomResourceDefinition {
	if self == nil {
		return nil
	}
	return &core.CustomResourceDefinition{
		Group:        self.Group,
		Version:      self.Version,
		Kind:         self.Kind,
		Plural:       self.Plural,
		Schema:       self.Schema,
		SpecMessage:  self.SpecMessage,
		ActionModule: self.ActionModule,
		Metadata:     self.Metadata.Convert(nil),
	}
}
