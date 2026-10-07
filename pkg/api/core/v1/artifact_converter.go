package corev1

import (
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/fieldmaskcommander"
)

func (self *ArtifactStorePath) Convert() core.ArtifactStorePath {
	if self == nil {
		return core.ArtifactStorePath{}
	}
	return core.ArtifactStorePath{
		EvalRef: self.GetEvalRef(),
	}
}

func (self *ArtifactSpecification) Convert() core.ArtifactSpecification {
	if self == nil {
		return core.ArtifactSpecification{}
	}
	return core.ArtifactSpecification{
		Store:     self.GetStore(),
		StorePath: self.GetStorePath().Convert(),
	}
}

func (self *ArtifactStatusPhase) Convert() core.ArtifactPhase {
	if self == nil {
		return core.ArtifactPhaseUnspecified
	}
	return core.ArtifactPhase(*self)
}

func (self *ArtifactStatus) Convert() core.ArtifactStatus {
	if self == nil {
		return core.ArtifactStatus{}
	}
	return core.ArtifactStatus{
		Phase:              self.Phase.Convert(),
		StorePath:          self.GetStorePath(),
		ObservedGeneration: self.GetObservedGeneration(),
		Message:            self.GetMessage(),
	}
}

func (self *Artifact) Convert(fmc *fieldmaskcommander.FieldMaskCommander) *core.Artifact {
	if self == nil {
		return nil
	}
	conversion := &core.Artifact{}
	conversion.APIVersion = self.GetApiVersion()
	conversion.Kind = self.GetKind()
	metadataFmc := fmc.GoTo("metadata")
	conversion.Metadata = self.Metadata.Convert(metadataFmc)
	conversion.Spec = self.Spec.Convert()
	conversion.Status = self.Status.Convert()
	return conversion
}
