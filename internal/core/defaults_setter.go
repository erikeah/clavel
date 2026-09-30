package core

import (
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
)

func SetDefaults_Metadata(m *Metadata) error {
	if m == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if m.CreationTimestamp == nil {
		now := time.Now().UTC()
		m.CreationTimestamp = &now
		// HINT: Following has been set because the resource is new
		m.Generation = -1
		m.ResourceVersion = ""
	}
	return nil
}

func SetDefaults_Evaluation(p *Evaluation) error {
	if p == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if p.Metadata == nil {
		p.Metadata = &Metadata{}
	}
	if err := SetDefaults_Metadata(p.Metadata); err != nil {
		return err
	}
	if p.Spec == nil {
		p.Spec = &EvaluationSpecification{}
	}
	if err := SetDefaults_EvaluationSpecification(p.Spec); err != nil {
		return err
	}
	return nil
}

func SetDefaults_EvaluationSpecification(spec *EvaluationSpecification) error {
	if spec == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	return nil
}
