package project

import (
	"errors"
	"regexp"
)

func ValidateFlakeref(flakeRef string) error {
	var flakerefErrors error
	if len(flakeRef) == 0 {
		flakerefErrors = errors.Join(flakerefErrors, errors.New("Flakeref cannot be empty"))
	}
	if regexp.MustCompile(`[ \t]`).MatchString(flakeRef) {
		flakerefErrors = errors.Join(flakerefErrors, errors.New("Flakeref cannot contain spaces or tabs"))
	}
	return flakerefErrors
}

func ValidateProjectSpecification(projectSpec *ProjectSpecification) error {
	var projectSpecErrors error
	err := ValidateFlakeref(projectSpec.Flakeref)
	if err != nil {
		projectSpecErrors = errors.Join(projectSpecErrors, err)
	}
	return projectSpecErrors
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

func ValidateProject(project *Project) error {
	var projectErrors error
	if err := ValidateName(project.Name); err != nil {
		projectErrors = errors.Join(projectErrors, err)
	}
	if err := ValidateProjectSpecification(project.Spec); err != nil {
		projectErrors = errors.Join(projectErrors, err)
	}
	return projectErrors
}
