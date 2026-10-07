package corev1

import (
	"github.com/erikeah/clavel/internal/core"
)

func (self *ArtifactStorePath) Set(storePath *core.ArtifactStorePath) {
	if storePath == nil {
		return
	}
	self.EvalRef = storePath.EvalRef
}

func (self *ArtifactSpecification) Set(spec *core.ArtifactSpecification) {
	if spec == nil {
		return
	}
	self.Store = spec.Store
	if self.StorePath == nil {
		self.StorePath = &ArtifactStorePath{}
	}
	self.StorePath.Set(&spec.StorePath)
}

func (self *ArtifactStatus) Set(status *core.ArtifactStatus) {
	if status == nil {
		return
	}
	self.Phase = ArtifactStatusPhase(status.Phase)
	self.StorePath = status.StorePath
	self.ObservedGeneration = status.ObservedGeneration
	self.Message = status.Message
}

func (self *Artifact) Set(p *core.Artifact) {
	if p == nil {
		return
	}
	self.ApiVersion = p.APIVersion
	self.Kind = p.Kind
	if self.Spec == nil {
		self.Spec = &ArtifactSpecification{}
	}
	self.Spec.Set(&p.Spec)
	if self.Status == nil {
		self.Status = &ArtifactStatus{}
	}
	self.Status.Set(&p.Status)
	if self.Metadata == nil {
		self.Metadata = &Metadata{}
	}
	self.Metadata.Set(&p.Metadata)
}
