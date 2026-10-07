package controller

import (
	"slices"
	"sync"
	"testing"
)

func TestDependencyTrackerFansOutToDependents(t *testing.T) {
	tracker := NewDependencyTracker()
	tracker.Track("artifact-a", []string{"eval"})
	tracker.Track("artifact-b", []string{"eval"})

	dependents := tracker.Dependents("eval")
	slices.Sort(dependents)
	if !slices.Equal(dependents, []string{"artifact-a", "artifact-b"}) {
		t.Fatalf("Dependents() = %v, want both artifacts", dependents)
	}
}

func TestDependencyTrackerRepPointingDropsTheOldEdge(t *testing.T) {
	tracker := NewDependencyTracker()
	tracker.Track("artifact", []string{"eval-old"})
	tracker.Track("artifact", []string{"eval-new"})

	if dependents := tracker.Dependents("eval-old"); len(dependents) != 0 {
		t.Fatalf("Dependents(eval-old) = %v, want none after re-pointing", dependents)
	}
	if dependents := tracker.Dependents("eval-new"); !slices.Equal(dependents, []string{"artifact"}) {
		t.Fatalf("Dependents(eval-new) = %v, want the artifact", dependents)
	}
}

func TestDependencyTrackerUntrackClearsEveryEdge(t *testing.T) {
	tracker := NewDependencyTracker()
	tracker.Track("artifact", []string{"eval-a", "eval-b"})
	tracker.Untrack("artifact")

	if tracker.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after untracking", tracker.Len())
	}
	if dependents := tracker.Dependents("eval-a"); len(dependents) != 0 {
		t.Fatalf("Dependents(eval-a) = %v, want none", dependents)
	}
}

func TestDependencyTrackerRepeatedRegistrationIsANoop(t *testing.T) {
	tracker := NewDependencyTracker()
	tracker.Track("artifact", []string{"eval"})
	tracker.Track("artifact", []string{"eval"})

	if dependents := tracker.Dependents("eval"); !slices.Equal(dependents, []string{"artifact"}) {
		t.Fatalf("Dependents(eval) = %v, want exactly one entry", dependents)
	}
	if tracker.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", tracker.Len())
	}
}

func TestDependencyTrackerNoDependentsReportsNil(t *testing.T) {
	tracker := NewDependencyTracker()
	if dependents := tracker.Dependents("nobody"); dependents != nil {
		t.Fatalf("Dependents(nobody) = %v, want nil", dependents)
	}
}

// Tracking from one goroutine while fanning out from another is exactly how a
// controller behaves: the artifact informer writes edges as artifacts arrive
// while the evaluation informer reads them.
func TestDependencyTrackerConcurrentAccess(t *testing.T) {
	tracker := NewDependencyTracker()
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dependent := "artifact-" + string(rune('a'+i))
			tracker.Track(dependent, []string{"eval"})
			tracker.Dependents("eval")
			tracker.Untrack(dependent)
		}()
	}
	wg.Wait()
	if tracker.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 after all dependents untracked", tracker.Len())
	}
}
