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

func TestWorkQueueRateLimitedRequeuedWhileProcessing(t *testing.T) {
	queue := NewWorkQueue(WithRetryBaseDelay(40*time.Millisecond), WithRetryMaxDelay(80*time.Millisecond))
	queue.Add("a")
	key, shutdown := queue.Get()
	if shutdown || key != "a" {
		t.Fatalf("expected key a, got %q shutdown=%v", key, shutdown)
	}
	// A failed reconcile requeues the key while it is still processing; the
	// retry must be delivered after the backoff, not dropped.
	queue.AddRateLimited("a")
	queue.Done("a")
	start := time.Now()
	next, shutdown := queue.Get()
	if shutdown || next != "a" {
		t.Fatalf("expected delayed requeue of a, got %q shutdown=%v", next, shutdown)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("rate limited key delivered too early: %s", elapsed)
	}
	queue.Done("a")
}

func TestWorkQueueAddAfterDedup(t *testing.T) {
	queue := NewWorkQueue()
	queue.AddAfter("a", time.Hour)
	queue.AddAfter("a", time.Hour)
	if queue.Len() != 1 {
		t.Fatalf("Len = %d, want 1 (second AddAfter for the same key is dropped)", queue.Len())
	}
	queue.AddAfter("b", 0)
	if queue.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (immediate AddAfter lands in the ready queue)", queue.Len())
	}
	key, shutdown := queue.Get()
	if shutdown || key != "b" {
		t.Fatalf("expected key b, got %q shutdown=%v", key, shutdown)
	}
	queue.Done("b")
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

func TestWorkQueueAddWakesGetBlockedOnDelayedItem(t *testing.T) {
	queue := NewWorkQueue()
	// A pending delayed item must not keep a worker asleep when a key becomes
	// ready: the Add signal used to be delivered to a goroutine sleeping on a
	// timer, where nothing was listening for it.
	queue.AddAfter("far", time.Hour)

	got := make(chan string, 1)
	go func() {
		key, _ := queue.Get()
		got <- key
	}()
	time.Sleep(50 * time.Millisecond)
	queue.Add("ready")

	select {
	case key := <-got:
		if key != "ready" {
			t.Fatalf("Get() = %q, want ready", key)
		}
	case <-time.After(time.Second):
		t.Fatal("Get() did not return a ready key while a delayed item was pending")
	}
	queue.Done("ready")
	queue.ShutDown()
}

func TestWorkQueuePromotesDelayedItemWithoutPollingGet(t *testing.T) {
	queue := NewWorkQueue(WithRetryBaseDelay(5*time.Millisecond), WithRetryMaxDelay(20*time.Millisecond))
	queue.AddAfter("later", 5*time.Millisecond)

	key, shutdown := queue.Get()
	if shutdown || key != "later" {
		t.Fatalf("Get() = %q shutdown=%v, want later", key, shutdown)
	}
	queue.Done("later")
	queue.ShutDown()
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
