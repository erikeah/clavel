package controller

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkQueueDedup(t *testing.T) {
	queue := NewWorkQueue()
	queue.Add("a")
	queue.Add("a")
	queue.Add("b")

	var processed sync.Map
	for {
		key, shutdown := queue.Get()
		if shutdown {
			break
		}
		if _, loaded := processed.LoadOrStore(key, true); loaded {
			t.Fatalf("key %q processed twice", key)
		}
		queue.Done(key)
		if syncEntries(&processed) == 2 {
			break
		}
	}
	if syncEntries(&processed) != 2 {
		t.Fatalf("expected 2 unique keys, got %d", syncEntries(&processed))
	}
}

func TestWorkQueueRateLimited(t *testing.T) {
	queue := NewWorkQueue(WithRetryBaseDelay(5*time.Millisecond), WithRetryMaxDelay(10*time.Millisecond))
	queue.AddRateLimited("a")
	start := time.Now()
	key, shutdown := queue.Get()
	if shutdown {
		t.Fatal("unexpected shutdown")
	}
	if key != "a" {
		t.Fatalf("expected key a, got %s", key)
	}
	// First retry should have been delayed by the base backoff.
	if elapsed := time.Since(start); elapsed < 4*time.Millisecond {
		t.Fatalf("rate limited item returned too quickly: %s", elapsed)
	}
	queue.Done(key)
}

func TestWorkQueueReaddWhileProcessing(t *testing.T) {
	queue := NewWorkQueue()
	queue.Add("a")
	key, _ := queue.Get()
	if key != "a" {
		t.Fatal("expected a")
	}
	queue.Add("a")
	queue.Done("a")
	next, shutdown := queue.Get()
	if shutdown || next != "a" {
		t.Fatalf("expected readded key a, got %q shutdown=%v", next, shutdown)
	}
	queue.Done("a")
}

func TestWorkQueueShutdown(t *testing.T) {
	queue := NewWorkQueue()
	queue.Add("a")
	queue.AddRateLimited("b")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, shutdown := queue.Get()
			if shutdown {
				return
			}
		}
	}()

	queue.ShutDown()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("queue did not drain on shutdown")
	}
}

func TestWorkQueueShutDownIdempotent(t *testing.T) {
	queue := NewWorkQueue()
	queue.ShutDown()
	queue.ShutDown()
	if !queue.ShuttingDown() {
		t.Fatal("expected queue to be shutting down")
	}
}

func syncEntries(m *sync.Map) int {
	var count int64
	m.Range(func(_, _ any) bool {
		atomic.AddInt64(&count, 1)
		return true
	})
	return int(count)
}
