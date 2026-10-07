package controller

import (
	"slices"
	"sync"
)

// DependencyTracker records which resources depend on a key, so a change to
// that key can be fanned out to them. The edges belong to the dependent and are
// replaced wholesale on every registration: a dependent that now points
// somewhere else stops being woken by the old dependency, and one that is gone
// stops being woken at all.
//
// It is what turns an upstream status change into requeued work. Without it a
// consumer can only poll its dependency, which retried it independently of the
// upstream instead of waiting for it.
type DependencyTracker struct {
	mu sync.RWMutex
	// dependentsOf is the reverse index: dependency key -> dependent keys.
	dependentsOf map[string]map[string]struct{}
	// dependsOn is kept so an edge can be removed without scanning the index,
	// and so a repeated registration can be recognised as a no-op.
	dependsOn map[string][]string
}

func NewDependencyTracker() *DependencyTracker {
	return &DependencyTracker{
		dependentsOf: make(map[string]map[string]struct{}),
		dependsOn:    make(map[string][]string),
	}
}

// Track declares that dependent depends on dependencies, replacing whatever it
// depended on before. Registering the same set again leaves the index alone.
func (t *DependencyTracker) Track(dependent string, dependencies []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if slices.Equal(t.dependsOn[dependent], dependencies) {
		return
	}
	for _, dependency := range t.dependsOn[dependent] {
		t.dropEdge(dependency, dependent)
	}
	if len(dependencies) == 0 {
		delete(t.dependsOn, dependent)
		return
	}
	t.dependsOn[dependent] = dependencies
	for _, dependency := range dependencies {
		set, ok := t.dependentsOf[dependency]
		if !ok {
			set = make(map[string]struct{})
			t.dependentsOf[dependency] = set
		}
		set[dependent] = struct{}{}
	}
}

// Untrack forgets everything dependent depended on.
func (t *DependencyTracker) Untrack(dependent string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, dependency := range t.dependsOn[dependent] {
		t.dropEdge(dependency, dependent)
	}
	delete(t.dependsOn, dependent)
}

// Dependents returns the keys that depend on dependency. The result is freshly
// built, so a caller may keep it across further updates.
func (t *DependencyTracker) Dependents(dependency string) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.dependentsOf[dependency]) == 0 {
		return nil
	}
	out := make([]string, 0, len(t.dependentsOf[dependency]))
	for dependent := range t.dependentsOf[dependency] {
		out = append(out, dependent)
	}
	return out
}

// Len reports how many dependencies currently have dependents, which is what a
// test wants to assert on rather than the map itself.
func (t *DependencyTracker) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.dependentsOf)
}

// dropEdge removes one edge. Callers hold t.mu.
func (t *DependencyTracker) dropEdge(dependency, dependent string) {
	set, ok := t.dependentsOf[dependency]
	if !ok {
		return
	}
	delete(set, dependent)
	if len(set) == 0 {
		delete(t.dependentsOf, dependency)
	}
}
