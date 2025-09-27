package core

import "time"

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
