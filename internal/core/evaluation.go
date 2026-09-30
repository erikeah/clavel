package core

import (
	"errors"
	"reflect"
	"regexp"

	"github.com/erikeah/clavel/internal/exceptions"
)

type EvaluationSpecification struct {
	Reference string `json:"reference"`
}

type EvaluationStatusResult struct {
	Hash      string `json:"hash,omitempty"`
	StorePath string `json:"storePath,omitempty"`
}

type EvaluationStatusPhase string

const (
	EvaluationStatusPhaseReady  EvaluationStatusPhase = "Ready"
	EvaluationStatusPhaseFailed EvaluationStatusPhase = "Failed"
)

type EvaluationStatus struct {
	Phase  EvaluationStatusPhase   `json:"phase,omitempty"`
	Result *EvaluationStatusResult `json:"result,omitempty"`
}

type Evaluation struct {
	Name     string                   `json:"name"`
	Metadata *Metadata                `json:"metadata"`
	Status   *EvaluationStatus        `json:"status,omitempty"`
	Spec     *EvaluationSpecification `json:"spec"`
}

func (p Evaluation) GetMetadataResourceVersion() string {
	return p.Metadata.GetResourceVersion()
}
func (p Evaluation) IncreaseMetadataGeneration() {
	p.Metadata.IncreaseGeneration()
}
func (p Evaluation) SetMetadataResourceVersion(rv string) {
	p.Metadata.SetResourceVersion(rv)
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

func MergeEvaluation(over, from *Evaluation) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if from.Name != "" {
		if from.Name != over.Name && over.Name != "" {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("Name cannot be changed"))
		}
		over.Name = from.Name
		hasChanged = true
	}
	if specHasChanged, err := MergeEvaluationSpecification(over.Spec, from.Spec); err != nil {
		return hasChanged, err
	} else if specHasChanged {
		defer over.IncreaseMetadataGeneration()
		hasChanged = true
	}
	if from.Status != nil {
		if !reflect.DeepEqual(over.Status, from.Status) {
			over.Status = from.Status
			hasChanged = true
		}
	}
	if metaHasChanged, err := MergeMetadata(over.Metadata, from.Metadata); err != nil {
		return hasChanged, err
	} else if metaHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}

func MergeEvaluationSpecification(over, from *EvaluationSpecification) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if over.Reference != from.Reference {
		over.Reference = from.Reference
		hasChanged = true
	}
	return hasChanged, nil
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

func ValidateEvaluationSpecification(evaluationSpec *EvaluationSpecification) error {
	var evaluationSpecErrors error
	err := ValidateReference(evaluationSpec.Reference)
	if err != nil {
		evaluationSpecErrors = errors.Join(evaluationSpecErrors, err)
	}
	return evaluationSpecErrors
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
