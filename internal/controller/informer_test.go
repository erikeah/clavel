package controller

import "testing"

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
