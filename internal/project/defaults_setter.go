package project

import (
	"github.com/erikeah/clavel/internal/core"
	"github.com/erikeah/clavel/internal/exceptions"
)

func SetDefaults_Project(p *Project) error {
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
		p.Spec = &ProjectSpecification{}
	}
	if err := SetDefaults_ProjectSpecification(p.Spec); err != nil {
		return err
	}
	return nil
}

func SetDefaults_ProjectSpecification(spec *ProjectSpecification) error {
	if spec == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	return nil
}
