package core

import (
	"errors"
	"regexp"
)

func ValidateMetadata(m *Metadata) error {
	var metadataErrors error
	if m.CreationTimestamp == nil {
		metadataErrors = errors.Join(metadataErrors, errors.New("CreationTimestamp cannot be empty"))
	}
	return metadataErrors
}

func ValidateReference(flakeRef string) error {
	var referenceErrors error
	if len(flakeRef) == 0 {
		referenceErrors = errors.Join(referenceErrors, errors.New("Reference cannot be empty"))
	}
	if regexp.MustCompile(`[ \t]`).MatchString(flakeRef) {
		referenceErrors = errors.Join(referenceErrors, errors.New("Reference cannot contain spaces or tabs"))
	}
	return referenceErrors
}

func ValidateEvaluationSpecification(evaluationSpec *EvaluationSpecification) error {
	var evaluationSpecErrors error
	err := ValidateReference(evaluationSpec.Reference)
	if err != nil {
		evaluationSpecErrors = errors.Join(evaluationSpecErrors, err)
	}
	return evaluationSpecErrors
}

func ValidateName(name string) error {
	var nameErrors error
	if len(name) == 0 {
		nameErrors = errors.Join(nameErrors, errors.New("Name cannot be empty"))
	}
	if regexp.MustCompile(`[ \t/]`).MatchString(name) {
		nameErrors = errors.Join(nameErrors, errors.New("Name cannot contain slashes, spaces or tabs"))
	}
	return nameErrors
}

func ValidateEvaluation(evaluation *Evaluation) error {
	var evaluationErrors error
	if err := ValidateName(evaluation.Name); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	if err := ValidateEvaluationSpecification(evaluation.Spec); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	if err := ValidateMetadata(evaluation.Metadata); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	return evaluationErrors
}
