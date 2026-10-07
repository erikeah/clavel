package controller

import (
	"slices"
	"strings"
	"testing"
)

type testResource struct {
	Name string
}

func newTestInformer() (*Informer[*testResource], WorkQueue) {
	queue := NewWorkQueue()
	informer := NewInformer[*testResource](queue, func(resource *testResource) string {
		return resource.Name
	})
	return informer, queue
}

func TestReplaceEvictsResourcesMissingFromTheSnapshot(t *testing.T) {
	informer, _ := newTestInformer()
	informer.Add(&testResource{Name: "removed"})
	informer.Add(&testResource{Name: "kept"})

	informer.Replace([]*testResource{{Name: "kept"}, {Name: "added"}})

	if _, ok := informer.Get("removed"); ok {
		t.Fatal("resource absent from the snapshot is still cached")
	}
	if _, ok := informer.Get("kept"); !ok {
		t.Fatal("resource present in the snapshot was dropped")
	}
	if _, ok := informer.Get("added"); !ok {
		t.Fatal("resource only present in the snapshot was not cached")
	}
}

func TestReplaceQueuesSnapshotAndEvictedKeys(t *testing.T) {
	informer, queue := newTestInformer()
	informer.Add(&testResource{Name: "removed"})
	informer.Add(&testResource{Name: "kept"})

	informer.Replace([]*testResource{{Name: "kept"}, {Name: "added"}})

	want := map[string]bool{"removed": true, "kept": true, "added": true}
	if queue.Len() != len(want) {
		t.Fatalf("queued keys = %d, want %d for a full resync", queue.Len(), len(want))
	}
	for range len(want) {
		key, shutdown := queue.Get()
		if shutdown {
			t.Fatal("queue shut down while draining")
		}
		if !want[key] {
			t.Fatalf("queued %q, want only snapshot and evicted keys", key)
		}
		delete(want, key)
		queue.Done(key)
	}
	if queue.Len() != 0 {
		t.Fatalf("queued keys = %d after draining, want 0", queue.Len())
	}
}

// drainKeys removes n keys from the queue and returns them sorted, so a test
// can compare against an expected set without depending on arrival order.
func drainKeys(queue WorkQueue, n int) []string {
	keys := make([]string, 0, n)
	for range n {
		key, shutdown := queue.Get()
		if shutdown {
			break
		}
		keys = append(keys, key)
		queue.Done(key)
	}
	slices.Sort(keys)
	return keys
}

func TestInformerWithoutHandlerEnqueuesTheChangedKey(t *testing.T) {
	informer, queue := newTestInformer()
	informer.Add(&testResource{Name: "upstream"})

	if keys := drainKeys(queue, queue.Len()); !slices.Equal(keys, []string{"upstream"}) {
		t.Fatalf("queued %v, want [upstream]", keys)
	}
}

// A trigger informer watches resources it does not reconcile: a change to the
// upstream is what wakes whatever depends on it.
func TestInformerWithOnChangeEnqueuesTheDependentsInstead(t *testing.T) {
	queue := NewWorkQueue()
	tracker := NewDependencyTracker()
	tracker.Track("artifact", []string{"upstream"})
	informer := NewInformer[*testResource](queue, func(resource *testResource) string {
		return resource.Name
	}, WithOnChange[*testResource](func(key string, resource *testResource, present bool) []string {
		return tracker.Dependents(key)
	}))

	informer.Add(&testResource{Name: "upstream"})

	if keys := drainKeys(queue, queue.Len()); !slices.Equal(keys, []string{"artifact"}) {
		t.Fatalf("queued %v, want the dependent artifact", keys)
	}
}

// The resync on reconnect must reach the handler too: the index it maintains
// would otherwise go stale for resources that only ever arrive in a snapshot.
func TestInformerWithOnChangeOffersSnapshotAndEvictedKeysToTheHandler(t *testing.T) {
	queue := NewWorkQueue()
	type change struct {
		key     string
		present bool
	}
	var changes []change
	informer := NewInformer[*testResource](queue, func(resource *testResource) string {
		return resource.Name
	}, WithOnChange[*testResource](func(key string, resource *testResource, present bool) []string {
		changes = append(changes, change{key: key, present: present})
		return []string{key}
	}))

	informer.Add(&testResource{Name: "removed"})
	informer.Add(&testResource{Name: "kept"})
	changes = nil
	informer.Replace([]*testResource{{Name: "kept"}, {Name: "added"}})

	slices.SortFunc(changes, func(a, b change) int {
		return strings.Compare(a.key, b.key)
	})
	want := []change{
		{key: "added", present: true},
		{key: "kept", present: true},
		{key: "removed", present: false},
	}
	if len(changes) != len(want) {
		t.Fatalf("handler saw %d changes, want %d: %+v", len(changes), len(want), changes)
	}
	for i := range want {
		if changes[i].key != want[i].key || changes[i].present != want[i].present {
			t.Fatalf("handler saw %+v, want %+v", changes[i], want[i])
		}
	}
	if keys := drainKeys(queue, queue.Len()); !slices.Equal(keys, []string{"added", "kept", "removed"}) {
		t.Fatalf("queued %v, want every snapshot and evicted key", keys)
	}
}
