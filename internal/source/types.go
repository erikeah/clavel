package source

import "github.com/erikeah/clavel/internal/core"

type SourceSpecification struct {
	Reference string `json:"reference"`
}

type Source struct {
	Name     string               `json:"name"`
	Metadata *core.Metadata       `json:"metadata"`
	Spec     *SourceSpecification `json:"spec"`
}

func (p Source) GetMetadataResourceVersion() string {
	return p.Metadata.GetResourceVersion()
}
func (p Source) IncreaseMetadataGeneration() {
	p.Metadata.IncreaseGeneration()
}
func (p Source) SetMetadataResourceVersion(rv string) {
	p.Metadata.SetResourceVersion(rv)
}
