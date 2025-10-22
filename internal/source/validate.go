package source

import (
	"errors"
	"regexp"

	"github.com/erikeah/clavel/internal/core"
)

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

func ValidateSourceSpecification(sourceSpec *SourceSpecification) error {
	var sourceSpecErrors error
	err := ValidateReference(sourceSpec.Reference)
	if err != nil {
		sourceSpecErrors = errors.Join(sourceSpecErrors, err)
	}
	return sourceSpecErrors
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

func ValidateSource(source *Source) error {
	var sourceErrors error
	if err := ValidateName(source.Name); err != nil {
		sourceErrors = errors.Join(sourceErrors, err)
	}
	if err := ValidateSourceSpecification(source.Spec); err != nil {
		sourceErrors = errors.Join(sourceErrors, err)
	}
	if err := core.ValidateMetadata(source.Metadata); err != nil {
		sourceErrors = errors.Join(sourceErrors, err)
	}
	return sourceErrors
}
