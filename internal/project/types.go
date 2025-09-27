package project

import "github.com/erikeah/clavel/internal/core"

type ProjectSpecification struct {
	Flakeref string `json:"flakeref"`
}

type Project struct {
	Name     string                `json:"name"`
	Metadata *core.Metadata        `json:"metadata"`
	Spec     *ProjectSpecification `json:"spec"`
}

func (p Project) GetMetadataResourceVersion() string {
	return p.Metadata.GetResourceVersion()
}
func (p Project) IncreaseMetadataGeneration() {
	p.Metadata.IncreaseGeneration()
}
func (p Project) SetMetadataResourceVersion(rv string) {
	p.Metadata.SetResourceVersion(rv)
}
