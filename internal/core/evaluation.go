package core

import (
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"

	"github.com/erikeah/clavel/internal/exceptions"
)

type EvaluationSpecification struct {
	Reference string `json:"reference"`
}

type EvaluationPhase int

const (
	EvaluationPhaseUnspecified EvaluationPhase = iota
	EvaluationPhaseSucceeded
	EvaluationPhaseFailed
	EvaluationPhasePending
)

type EvaluationStatus struct {
	Phase EvaluationPhase `json:"phase"`
	// Result holds the raw JSON evaluation result base64-encoded, so the
	// stored and transmitted representation needs no escaping.
	Result             string `json:"result,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration"`
	Message            string `json:"message,omitempty"`
}

// EncodeResult base64-encodes a raw JSON evaluation result for storage and
// transmission.
func EncodeResult(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
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

func ValidateEvaluationSpecification(evaluationSpec EvaluationSpecification) error {
	var evaluationSpecErrors error
	if err := ValidateReference(evaluationSpec.Reference); err != nil {
		evaluationSpecErrors = errors.Join(evaluationSpecErrors, err)
	}
	return evaluationSpecErrors
}

func SetDefaults_EvaluationSpecification(spec *EvaluationSpecification) error {
	if spec == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	return nil
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

// TypeMeta of an Evaluation as stored; the versioned schema of every resource.
const (
	EvaluationAPIVersion = "clavel.core/v1"
	EvaluationKind       = "Evaluation"
)

type Evaluation struct {
	APIVersion string                  `json:"apiVersion"`
	Kind       string                  `json:"kind"`
	Metadata   Metadata                `json:"metadata"`
	Status     EvaluationStatus        `json:"status"`
	Spec       EvaluationSpecification `json:"spec"`
}

func (p *Evaluation) GetMetadataResourceVersion() string {
	if p == nil {
		return ""
	}
	return p.Metadata.GetResourceVersion()
}

func (p *Evaluation) IncreaseMetadataGeneration() {
	if p == nil {
		return
	}
	p.Metadata.IncreaseGeneration()
}

func (p *Evaluation) SetMetadataResourceVersion(rv string) {
	if p == nil {
		return
	}
	p.Metadata.SetResourceVersion(rv)
}

func ValidateEvaluationStatus(status EvaluationStatus) error {
	var statusErrors error
	switch status.Phase {
	case EvaluationPhaseUnspecified, EvaluationPhaseSucceeded, EvaluationPhaseFailed, EvaluationPhasePending:
	default:
		statusErrors = errors.Join(statusErrors, errors.New("Phase is not valid"))
	}
	return statusErrors
}

func ValidateTypeMeta(apiVersion string, kind string) error {
	var typeMetaErrors error
	if apiVersion != EvaluationAPIVersion {
		typeMetaErrors = errors.Join(typeMetaErrors, fmt.Errorf("APIVersion must be %q", EvaluationAPIVersion))
	}
	if kind != EvaluationKind {
		typeMetaErrors = errors.Join(typeMetaErrors, fmt.Errorf("Kind must be %q", EvaluationKind))
	}
	return typeMetaErrors
}

func ValidateEvaluation(evaluation Evaluation) error {
	var evaluationErrors error
	if err := ValidateTypeMeta(evaluation.APIVersion, evaluation.Kind); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	if err := ValidateMetadata(&evaluation.Metadata); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	if err := ValidateEvaluationSpecification(evaluation.Spec); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	if err := ValidateEvaluationStatus(evaluation.Status); err != nil {
		evaluationErrors = errors.Join(evaluationErrors, err)
	}
	return evaluationErrors
}

func SetDefaults_Evaluation(p *Evaluation) error {
	if p == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if p.APIVersion == "" {
		p.APIVersion = EvaluationAPIVersion
	}
	if p.Kind == "" {
		p.Kind = EvaluationKind
	}
	if err := SetDefaults_Metadata(&p.Metadata); err != nil {
		return err
	}
	if err := SetDefaults_EvaluationSpecification(&p.Spec); err != nil {
		return err
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
	// TypeMeta is part of the requested identity; it is validated, never
	// silently rewritten.
	if from.APIVersion != "" && from.APIVersion != over.APIVersion {
		over.APIVersion = from.APIVersion
		hasChanged = true
	}
	if from.Kind != "" && from.Kind != over.Kind {
		over.Kind = from.Kind
		hasChanged = true
	}
	if specHasChanged, err := MergeEvaluationSpecification(&over.Spec, &from.Spec); err != nil {
		return hasChanged, err
	} else if specHasChanged {
		defer over.IncreaseMetadataGeneration()
		hasChanged = true
	}
	if metaHasChanged, err := MergeMetadata(&over.Metadata, &from.Metadata); err != nil {
		return hasChanged, err
	} else if metaHasChanged {
		hasChanged = true
	}
	if statusHasChanged, err := MergeEvaluationStatus(&over.Status, from.Status); err != nil {
		return hasChanged, err
	} else if statusHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}

func MergeEvaluationStatus(over *EvaluationStatus, from EvaluationStatus) (bool, error) {
	if over == nil {
		// TODO: Logging or sensible error
		return false, exceptions.InternalFailure
	}
	if from == (EvaluationStatus{}) {
		return false, nil
	}
	if *over == from {
		return false, nil
	}
	*over = from
	return true, nil
}
