package corev1

import (
	"encoding/json"

	"github.com/erikeah/clavel/internal/core"
)

func (self *Resource) Set(r *core.Resource) {
	if r == nil {
		return
	}
	self.Name = r.Name
	self.Group = r.Group
	self.Version = r.Version
	self.Plural = r.Plural
	self.Kind = r.Kind
	self.Spec = json.RawMessage(r.Spec)
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(r.Metadata)
}
