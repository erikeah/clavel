package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/erikeah/clavel/internal/exceptions"
)

type ArtifactStorePath struct {
	EvalRef string `json:"evalRef"`
}

type ArtifactSpecification struct {
	Store     string            `json:"store"`
	StorePath ArtifactStorePath `json:"storePath"`
}

type ArtifactPhase int

const (
	ArtifactPhaseUnspecified ArtifactPhase = iota
	ArtifactPhasePending
	ArtifactPhaseSucceeded
	ArtifactPhaseFailed
)

type ArtifactStatus struct {
	Phase ArtifactPhase `json:"phase"`
	// StorePath is the concrete path the evalRef resolved to. It is recorded so
	// a consumer can read the resolved reference without re-running the
	// evaluation, even after the spec has changed.
	StorePath          string `json:"storePath,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration"`
	Message            string `json:"message,omitempty"`
}

// StorePathFromResult extracts the store path from an evaluation result. The
// result is the base64-encoded JSON an Evaluation wrote to status.result;
// anything else is a contract violation rather than a path to hand to nix.
func StorePathFromResult(result string) (string, error) {
	if result == "" {
		return "", errors.Join(exceptions.InvalidArguments, errors.New("evaluation has no result"))
	}
	raw, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return "", errors.Join(exceptions.InvalidArguments, fmt.Errorf("evaluation result is not base64: %w", err))
	}
	var decoded struct {
		StorePath string `json:"storePath"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", errors.Join(exceptions.InvalidArguments, fmt.Errorf("evaluation result is not a JSON object: %w", err))
	}
	if decoded.StorePath == "" {
		return "", errors.Join(exceptions.InvalidArguments, errors.New("evaluation result has no storePath"))
	}
	if len(decoded.StorePath) < len("/nix/store/") || decoded.StorePath[:len("/nix/store/")] != "/nix/store/" {
		return "", errors.Join(exceptions.InvalidArguments, fmt.Errorf("storePath %q is not a nix store path", decoded.StorePath))
	}
	return decoded.StorePath, nil
}

// TypeMeta of an Artifact as stored; the versioned schema of every resource.
const (
	ArtifactAPIVersion = "clavel.core/v1"
	ArtifactKind       = "Artifact"
)

type Artifact struct {
	APIVersion string                `json:"apiVersion"`
	Kind       string                `json:"kind"`
	Metadata   Metadata              `json:"metadata"`
	Status     ArtifactStatus        `json:"status"`
	Spec       ArtifactSpecification `json:"spec"`
}

func (p *Artifact) GetMetadataResourceVersion() string {
	if p == nil {
		return ""
	}
	return p.Metadata.GetResourceVersion()
}

func (p *Artifact) IncreaseMetadataGeneration() {
	if p == nil {
		return
	}
	p.Metadata.IncreaseGeneration()
}

func (p *Artifact) SetMetadataResourceVersion(rv string) {
	if p == nil {
		return
	}
	p.Metadata.SetResourceVersion(rv)
}

// ValidateArtifactSpecification checks the reference an artifact resolves.
// Store is optional: an empty store is the local store, which is what a path
// already known to the machine resolves against.
func ValidateArtifactSpecification(spec ArtifactSpecification) error {
	var specErrors error
	if len(spec.StorePath.EvalRef) == 0 {
		specErrors = errors.Join(specErrors, errors.New("StorePath.EvalRef cannot be empty"))
	} else if err := ValidateName(spec.StorePath.EvalRef); err != nil {
		specErrors = errors.Join(specErrors, fmt.Errorf("StorePath.EvalRef: %w", err))
	}
	return specErrors
}

func ValidateArtifactStatus(status ArtifactStatus) error {
	var statusErrors error
	switch status.Phase {
	case ArtifactPhaseUnspecified, ArtifactPhasePending, ArtifactPhaseSucceeded, ArtifactPhaseFailed:
	default:
		statusErrors = errors.Join(statusErrors, errors.New("Phase is not valid"))
	}
	return statusErrors
}

func ValidateArtifactTypeMeta(apiVersion string, kind string) error {
	var typeMetaErrors error
	if apiVersion != ArtifactAPIVersion {
		typeMetaErrors = errors.Join(typeMetaErrors, fmt.Errorf("APIVersion must be %q", ArtifactAPIVersion))
	}
	if kind != ArtifactKind {
		typeMetaErrors = errors.Join(typeMetaErrors, fmt.Errorf("Kind must be %q", ArtifactKind))
	}
	return typeMetaErrors
}

func ValidateArtifact(artifact Artifact) error {
	var artifactErrors error
	if err := ValidateArtifactTypeMeta(artifact.APIVersion, artifact.Kind); err != nil {
		artifactErrors = errors.Join(artifactErrors, err)
	}
	if err := ValidateMetadata(&artifact.Metadata); err != nil {
		artifactErrors = errors.Join(artifactErrors, err)
	}
	if err := ValidateArtifactSpecification(artifact.Spec); err != nil {
		artifactErrors = errors.Join(artifactErrors, err)
	}
	if err := ValidateArtifactStatus(artifact.Status); err != nil {
		artifactErrors = errors.Join(artifactErrors, err)
	}
	return artifactErrors
}

func SetDefaults_ArtifactSpecification(spec *ArtifactSpecification) error {
	if spec == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	return nil
}

func SetDefaults_Artifact(p *Artifact) error {
	if p == nil {
		// TODO: Logging or sensible error
		return exceptions.InternalFailure
	}
	if p.APIVersion == "" {
		p.APIVersion = ArtifactAPIVersion
	}
	if p.Kind == "" {
		p.Kind = ArtifactKind
	}
	if err := SetDefaults_Metadata(&p.Metadata); err != nil {
		return err
	}
	if err := SetDefaults_ArtifactSpecification(&p.Spec); err != nil {
		return err
	}
	return nil
}

func MergeArtifactSpecification(over, from *ArtifactSpecification) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, nil
	}
	if over.Store != from.Store {
		over.Store = from.Store
		hasChanged = true
	}
	if over.StorePath.EvalRef != from.StorePath.EvalRef {
		over.StorePath.EvalRef = from.StorePath.EvalRef
		hasChanged = true
	}
	return hasChanged, nil
}

func MergeArtifactStatus(over *ArtifactStatus, from ArtifactStatus) (bool, error) {
	if over == nil {
		// TODO: Logging or sensible error
		return false, exceptions.InternalFailure
	}
	if from == (ArtifactStatus{}) {
		return false, nil
	}
	if *over == from {
		return false, nil
	}
	*over = from
	return true, nil
}

func MergeArtifact(over, from *Artifact) (bool, error) {
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
	if specHasChanged, err := MergeArtifactSpecification(&over.Spec, &from.Spec); err != nil {
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
	if statusHasChanged, err := MergeArtifactStatus(&over.Status, from.Status); err != nil {
		return hasChanged, err
	} else if statusHasChanged {
		hasChanged = true
	}
	return hasChanged, nil
}
