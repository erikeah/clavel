package source

import (
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/exceptions"
)

func SetDefaults_Source(p *Source) error {
	if p == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if p.Metadata == nil {
		p.Metadata = &core.Metadata{}
	}
	if err := core.SetDefaults_Metadata(p.Metadata); err != nil {
		return err
	}
	if p.Spec == nil {
		p.Spec = &SourceSpecification{}
	}
	if err := SetDefaults_SourceSpecification(p.Spec); err != nil {
		return err
	}
	return nil
}

func SetDefaults_SourceSpecification(spec *SourceSpecification) error {
	if spec == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	return nil
}
