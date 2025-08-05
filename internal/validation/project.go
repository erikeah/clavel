package validation

import (
	"fmt"
	"regexp"

	"github.com/erikeah/clavel/internal/errors"
	"github.com/erikeah/clavel/internal/model"
)

func ValidateFlakeref(flakeRef string) errors.Error {
	var flakerefErrors errors.Error
	if len(flakeRef) == 0 {
		flakerefErrors = errors.Join(flakerefErrors, errors.New(errors.BadCallingParameters, "Flakeref cannot be empty"))
	}
	if regexp.MustCompile(`[ \t]`).MatchString(flakeRef) {
		flakerefErrors = errors.Join(flakerefErrors, errors.New(errors.BadCallingParameters, "Flakeref cannot contain spaces or tabs"))
	}
	return flakerefErrors
}

func ValidateProjectSpecification(projectSpec *model.ProjectSpecification) errors.Error {
	var projectSpecErrors errors.Error
	err := ValidateFlakeref(projectSpec.Flakeref)
	if err != nil {
		projectSpecErrors = errors.Join(projectSpecErrors, err)
	}
	return projectSpecErrors
}

func ValidateProject(project *model.Project) errors.Error {
	var projectErrors errors.Error
	if len(project.Name) == 0 {
		projectErrors = errors.Join(projectErrors, errors.New(errors.BadCallingParameters, "Name cannot be empty"))
	}
	if regexp.MustCompile(`[ \t/]`).MatchString(project.Name) {
		projectErrors = errors.Join(projectErrors, errors.New(errors.BadCallingParameters, "Name cannot contain slashes, spaces or tabs"))
	}
	if err := ValidateProjectSpecification(project.Spec); err != nil {
		projectErrors = errors.Join(projectErrors, errors.New(errors.BadCallingParameters, fmt.Sprintf("At Spec:\n%s", err.Error())))
	}
	return projectErrors
}
