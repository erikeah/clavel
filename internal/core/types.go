package core

import "time"

type EvaluationSpecification struct {
	Reference string `json:"reference"`
}

type Evaluation struct {
	Name     string                   `json:"name"`
	Metadata *Metadata                `json:"metadata"`
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

type Metadata struct {
	generationHasIncrease bool
	Generation            int64      `json:"generation"`
	ResourceVersion       string     `json:"-"`
	Finalizers            []string   `json:"finalizers,omitempty"`
	CreationTimestamp     *time.Time `json:"creationTimestamp"`
	DeletionTimestamp     *time.Time `json:"deletionTimestamp,omitempty"`
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
