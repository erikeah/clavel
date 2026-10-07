package genericstore

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type testResource struct {
	Name            string `json:"name"`
	ResourceVersion string `json:"-"`
}

func (r *testResource) GetMetadataResourceVersion() string   { return r.ResourceVersion }
func (r *testResource) SetMetadataResourceVersion(rv string) { r.ResourceVersion = rv }

// newTestStore connects to the etcd the dev services provide and returns a
// store under a unique key space. It skips instead of failing when etcd is not
// reachable, so the suite stays runnable without the dev environment.
func newTestStore(t *testing.T) *store[*testResource] {
	t.Helper()
	endpoint := os.Getenv("CLAVEL_TEST_ETCD")
	if endpoint == "" {
		endpoint = "127.0.0.1:2379"
	}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Skipf("etcd unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := client.Status(ctx, endpoint); err != nil {
		client.Close()
		t.Skipf("etcd unavailable at %s: %v", endpoint, err)
	}
	st := NewStore[*testResource](client, []string{
		"genericstore-test", strings.ReplaceAll(t.Name(), "/", "-"),
	})
	t.Cleanup(func() {
		deleteCtx, deleteCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer deleteCancel()
		client.Delete(deleteCtx, st.prefix(), clientv3.WithPrefix())
		client.Close()
	})
	if _, err := client.Delete(ctx, st.prefix(), clientv3.WithPrefix()); err != nil {
		t.Fatalf("seeding cleanup failed: %v", err)
	}
	return st
}

func nextEvent(t *testing.T, events <-chan Event[*testResource], errs <-chan error) Event[*testResource] {
	t.Helper()
	select {
	case event, ok := <-events:
		if !ok {
			t.Fatal("watch channel closed")
		}
		return event
	case err := <-errs:
		t.Fatalf("watch failed: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a watch event")
	}
	return Event[*testResource]{}
}

func TestWatchSnapshotsThenStreamsChanges(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	if err := st.Create(ctx, "existing", &testResource{Name: "existing"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	events, errs := st.Watch(ctx, true)

	snapshot := nextEvent(t, events, errs)
	if snapshot.Type != EventTypePut || snapshot.Name != "existing" {
		t.Fatalf("first event = %+v, want a put of existing", snapshot)
	}
	if snapshot.Object.Name != "existing" {
		t.Fatalf("object = %+v, want the stored resource", snapshot.Object)
	}
	if snapshot.Object.ResourceVersion == "" {
		t.Fatal("snapshot put carries no resourceVersion")
	}
	if sync := nextEvent(t, events, errs); sync.Type != EventTypeSync {
		t.Fatalf("second event = %+v, want the sync marker", sync)
	}

	if err := st.Create(ctx, "live", &testResource{Name: "live"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created := nextEvent(t, events, errs); created.Type != EventTypePut || created.Name != "live" {
		t.Fatalf("event = %+v, want a put of live", created)
	}

	if err := st.Delete(ctx, "existing"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	removed := nextEvent(t, events, errs)
	if removed.Type != EventTypeDelete || removed.Name != "existing" {
		t.Fatalf("event = %+v, want a delete of existing", removed)
	}
	if removed.Object != nil {
		t.Fatalf("object = %+v, want none: a delete carries only the identity", removed.Object)
	}
}

// An empty snapshot must still be terminated: the informer replaces its cache
// with nothing on the sync marker, which is how resources deleted while the
// stream was down disappear from it.
func TestWatchEmptySnapshotStillSignalsSync(t *testing.T) {
	st := newTestStore(t)

	events, errs := st.Watch(context.Background(), true)

	if sync := nextEvent(t, events, errs); sync.Type != EventTypeSync {
		t.Fatalf("first event = %+v, want an immediate sync marker", sync)
	}
}

func TestWatchWithoutSnapshotHasNoSync(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	if err := st.Create(ctx, "seed", &testResource{Name: "seed"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	events, errs := st.Watch(ctx, false)

	if err := st.Delete(ctx, "seed"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	removed := nextEvent(t, events, errs)
	if removed.Type != EventTypeDelete || removed.Name != "seed" {
		t.Fatalf("event = %+v, want a delete of seed", removed)
	}
}
