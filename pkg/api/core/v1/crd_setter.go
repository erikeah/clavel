package corev1

import (
	"github.com/erikeah/clavel/internal/core"
)

func (self *CustomResourceDefinition) Set(c *core.CustomResourceDefinition) {
	if c == nil {
		return
	}
	self.Group = c.Group
	self.Version = c.Version
	self.Kind = c.Kind
	self.Plural = c.Plural
	self.Schema = c.Schema
	self.SpecMessage = c.SpecMessage
	self.ActionModule = c.ActionModule
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(c.Metadata)
}
