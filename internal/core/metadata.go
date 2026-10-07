package core

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/erikeah/clavel/internal/exceptions"
	"github.com/google/uuid"
)

// OwnerReference points at another resource this one depends on or is owned by;
// the owner set drives pruning and teardown ordering.
type OwnerReference struct {
	APIVersion         string `json:"apiVersion"`
	Kind               string `json:"kind"`
	Name               string `json:"name"`
	UID                string `json:"uid"`
	Controller         bool   `json:"controller,omitempty"`
	BlockOwnerDeletion bool   `json:"blockOwnerDeletion,omitempty"`
}

type Metadata struct {
	Name                  string `json:"name"`
	generationHasIncrease bool
	Generation            int64             `json:"generation"`
	ResourceVersion       string            `json:"-"`
	UID                   string            `json:"uid"`
	Labels                map[string]string `json:"labels,omitempty"`
	Annotations           map[string]string `json:"annotations,omitempty"`
	OwnerReferences       []OwnerReference  `json:"ownerReferences,omitempty"`
	GenerateName          string            `json:"generateName,omitempty"`
	Namespace             string            `json:"namespace,omitempty"`
	Finalizers            []string          `json:"finalizers,omitempty"`
	CreationTimestamp     *time.Time        `json:"creationTimestamp"`
	DeletionTimestamp     *time.Time        `json:"deletionTimestamp,omitempty"`
}

func (m *Metadata) IncreaseGeneration() {
	if m.generationHasIncrease {
		return
	}
	m.Generation++
	m.generationHasIncrease = true
}

func (m *Metadata) SetResourceVersion(resourceVersion string) {
	m.ResourceVersion = resourceVersion
}

func (m *Metadata) GetResourceVersion() string {
	if m == nil {
		return ""
	}
	return m.ResourceVersion
}

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
		if m.UID == "" {
			m.UID = uuid.NewString()
		}
	}
	return nil
}

var (
	nameRegexp       = regexp.MustCompile(`[ \t/]`)
	labelKeyRegexp   = regexp.MustCompile(`^([a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/)?[A-Za-z0-9]([-A-Za-z0-9_.]*[A-Za-z0-9])?$`)
	labelValueRegexp = regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)
)

// Label keys and values follow the k8s qualified-name limits: an optional DNS
// subdomain prefix, then at most 63 characters of name.
const (
	maxLabelKeyLen   = 253 + 1 + 63
	maxLabelValueLen = 63
)

func validateLabelKeys(values map[string]string, field string) error {
	var valuesErrors error
	for key, value := range values {
		if len(key) > maxLabelKeyLen || !labelKeyRegexp.MatchString(key) {
			valuesErrors = errors.Join(valuesErrors, fmt.Errorf("%s key %q is invalid", field, key))
			continue
		}
		if len(value) > maxLabelValueLen || !labelValueRegexp.MatchString(value) {
			valuesErrors = errors.Join(valuesErrors, fmt.Errorf("%s value for key %q is invalid", field, key))
		}
	}
	return valuesErrors
}

func validateOwnerReferences(references []OwnerReference) error {
	var referencesErrors error
	controllers := 0
	for i, reference := range references {
		if reference.APIVersion == "" || reference.Kind == "" || reference.Name == "" || reference.UID == "" {
			referencesErrors = errors.Join(referencesErrors, fmt.Errorf("ownerReference[%d] requires apiVersion, kind, name and uid", i))
		}
		if reference.Controller {
			controllers++
		}
	}
	if controllers > 1 {
		referencesErrors = errors.Join(referencesErrors, errors.New("at most one ownerReference may set controller"))
	}
	return referencesErrors
}

func ValidateMetadata(m *Metadata) error {
	if m == nil {
		return errors.New("Metadata cannot be nil")
	}
	var metadataErrors error
	if err := ValidateName(m.Name); err != nil {
		metadataErrors = errors.Join(metadataErrors, err)
	}
	if m.CreationTimestamp == nil {
		metadataErrors = errors.Join(metadataErrors, errors.New("CreationTimestamp cannot be empty"))
	}
	if len(m.UID) == 0 {
		metadataErrors = errors.Join(metadataErrors, errors.New("UID cannot be empty"))
	}
	if err := validateLabelKeys(m.Labels, "label"); err != nil {
		metadataErrors = errors.Join(metadataErrors, err)
	}
	if err := validateLabelKeys(m.Annotations, "annotation"); err != nil {
		metadataErrors = errors.Join(metadataErrors, err)
	}
	if err := validateOwnerReferences(m.OwnerReferences); err != nil {
		metadataErrors = errors.Join(metadataErrors, err)
	}
	return metadataErrors
}

func MergeMetadata(over, from *Metadata) (bool, error) {
	var hasChanged bool = false
	if over == nil {
		// TODO: Logging or sensible error
		return hasChanged, exceptions.InternalFailure
	}
	if from == nil {
		return hasChanged, exceptions.InvalidArguments
	}
	// Optimistic concurrency: the caller must send the revision it read, so a
	// write based on a stale read fails here and again, atomically, in the
	// store. A conflict is retryable, not a bad request.
	if over.ResourceVersion != from.ResourceVersion {
		return hasChanged, errors.Join(exceptions.Conflict, errors.New("resourceVersion does not match"))
	}
	// Name is the identity: settable only while unset, never renamed.
	if from.Name != "" {
		if over.Name != "" && from.Name != over.Name {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("Name cannot be changed"))
		}
		if over.Name != from.Name {
			over.Name = from.Name
			hasChanged = true
		}
	}
	// Uid is assigned by the server and never changes.
	if from.UID != "" && from.UID != over.UID {
		return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("UID cannot be changed"))
	}
	// Namespace and GenerateName are identity fields too: settable only while
	// unset, never changed afterwards.
	if from.Namespace != "" {
		if over.Namespace != "" && from.Namespace != over.Namespace {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("Namespace cannot be changed"))
		}
		if over.Namespace != from.Namespace {
			over.Namespace = from.Namespace
			hasChanged = true
		}
	}
	if from.GenerateName != "" {
		if over.GenerateName != "" && from.GenerateName != over.GenerateName {
			return hasChanged, errors.Join(exceptions.InvalidArguments, errors.New("GenerateName cannot be changed"))
		}
		if over.GenerateName != from.GenerateName {
			over.GenerateName = from.GenerateName
			hasChanged = true
		}
	}
	// HINT: Maps and slices are replaced as a whole when provided; merging them
	// per key is deferred with the field masking enhancement (see TODO.md).
	if from.Labels != nil {
		over.Labels = from.Labels
		hasChanged = true
	}
	if from.Annotations != nil {
		over.Annotations = from.Annotations
		hasChanged = true
	}
	if from.OwnerReferences != nil {
		over.OwnerReferences = from.OwnerReferences
		hasChanged = true
	}
	if from.Finalizers != nil {
		over.Finalizers = from.Finalizers
		hasChanged = true
	}
	if from.DeletionTimestamp != nil && (over.DeletionTimestamp == nil || from.DeletionTimestamp.After(*over.DeletionTimestamp)) {
		over.DeletionTimestamp = from.DeletionTimestamp
		hasChanged = true
	}
	// Generation and CreationTimestamp is not merged on purpose
	return hasChanged, nil
}

// ValidateName reports whether name is a usable resource identity.
func ValidateName(name string) error {
	var nameErrors error
	if len(name) == 0 {
		nameErrors = errors.Join(nameErrors, errors.New("Name cannot be empty"))
	}
	if nameRegexp.MatchString(name) {
		nameErrors = errors.Join(nameErrors, errors.New("Name cannot contain slashes, spaces or tabs"))
	}
	return nameErrors
}
