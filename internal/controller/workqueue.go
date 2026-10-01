package controller

import (
	"container/heap"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultRetryBaseDelay = 5 * time.Millisecond
	defaultRetryMaxDelay  = 1000 * time.Second
)

// WorkQueue is a rate-limited, deduplicating queue of string keys, mirroring
// the Kubernetes client-go workqueue semantics.
type WorkQueue interface {
	// Add enqueues a key. A key already waiting in the queue is not enqueued twice.
	Add(key string)
	// AddRateLimited enqueues a key after a per-key retry backoff and an optional
	// global rate limiter delay.
	AddRateLimited(key string)
	// Forget resets the retry/backoff bookkeeping for a key.
	Forget(key string)
	// Get returns the next key. The second return is true once the queue has been
	// shut down and drained.
	Get() (key string, shutdown bool)
	// Done marks a key returned by Get as finished processing. Keys re-added while
	// processing are re-enqueued here.
	Done(key string)
	Len() int
	ShuttingDown() bool
	ShutDown()
}

// WorkQueueOption configures a new WorkQueue.
type WorkQueueOption func(*workQueue)

// WithRateLimiter caps the overall requeue throughput between retry backoffs.
func WithRateLimiter(limiter *rate.Limiter) WorkQueueOption {
	return func(q *workQueue) { q.limiter = limiter }
}

func WithRetryBaseDelay(delay time.Duration) WorkQueueOption {
	return func(q *workQueue) { q.baseDelay = delay }
}

func WithRetryMaxDelay(delay time.Duration) WorkQueueOption {
	return func(q *workQueue) { q.maxDelay = delay }
}

type delayedItem struct {
	key     string
	readyAt time.Time
}

// delayedHeap orders delayed items by their ready time.
type delayedHeap []delayedItem

func (h delayedHeap) Len() int           { return len(h) }
func (h delayedHeap) Less(i, j int) bool { return h[i].readyAt.Before(h[j].readyAt) }
func (h delayedHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *delayedHeap) Push(x any) {
	*h = append(*h, x.(delayedItem))
}

func (h *delayedHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type workQueue struct {
	mu         sync.Mutex
	cond       *sync.Cond
	ready      []string
	delayed    delayedHeap
	dirty      map[string]bool
	processing map[string]bool
	retries    map[string]int

	limiter   *rate.Limiter
	baseDelay time.Duration
	maxDelay  time.Duration

	shutDown bool
}

func NewWorkQueue(options ...WorkQueueOption) WorkQueue {
	q := &workQueue{
		dirty:      make(map[string]bool),
		processing: make(map[string]bool),
		retries:    make(map[string]int),
		baseDelay:  defaultRetryBaseDelay,
		maxDelay:   defaultRetryMaxDelay,
	}
	q.cond = sync.NewCond(&q.mu)
	for _, option := range options {
		option(q)
	}
	return q
}

func (q *workQueue) Add(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.add(key)
}

func (q *workQueue) add(key string) {
	if q.shutDown {
		return
	}
	if q.dirty[key] {
		return
	}
	q.dirty[key] = true
	if q.processing[key] {
		// Re-added while processing; Done re-enqueues it.
		return
	}
	q.ready = append(q.ready, key)
	q.cond.Signal()
}

func (q *workQueue) AddRateLimited(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.shutDown || q.dirty[key] || q.processing[key] {
		return
	}
	q.retries[key]++
	delay := q.backoffDelay(key)
	q.dirty[key] = true
	if delay <= 0 {
		q.ready = append(q.ready, key)
	} else {
		heap.Push(&q.delayed, delayedItem{key: key, readyAt: time.Now().Add(delay)})
	}
	q.cond.Broadcast()
}

func (q *workQueue) Forget(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.retries, key)
}

func (q *workQueue) Get() (key string, shutdown bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	for {
		if len(q.ready) > 0 {
			key, q.ready = q.ready[0], q.ready[1:]
			delete(q.dirty, key)
			q.processing[key] = true
			return key, false
		}
		if len(q.delayed) > 0 && !q.delayed[0].readyAt.After(time.Now()) {
			item := heap.Pop(&q.delayed).(delayedItem)
			delete(q.dirty, item.key)
			q.processing[item.key] = true
			return item.key, false
		}
		if q.shutDown {
			return "", true
		}
		if len(q.delayed) > 0 {
			wait := time.Until(q.delayed[0].readyAt)
			if wait > 0 {
				timer := time.NewTimer(wait)
				q.mu.Unlock()
				<-timer.C
				q.mu.Lock()
				continue
			}
		}
		q.cond.Wait()
	}
}

func (q *workQueue) Done(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.processing[key] {
		return
	}
	delete(q.processing, key)
	if q.dirty[key] {
		q.ready = append(q.ready, key)
		q.cond.Signal()
	}
}

func (q *workQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.ready) + q.delayed.Len()
}

func (q *workQueue) ShutDown() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.shutDown {
		return
	}
	q.shutDown = true
	q.cond.Broadcast()
}

func (q *workQueue) ShuttingDown() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.shutDown
}

// backoffDelay computes the retry delay for a key: exponential backoff based on
// the requeue count, bounded by the global rate limiter when configured.
func (q *workQueue) backoffDelay(key string) time.Duration {
	backoff := q.baseDelay
	for i := 1; i < q.retries[key]; i++ {
		backoff *= 2
		if backoff >= q.maxDelay {
			backoff = q.maxDelay
			break
		}
	}
	if q.limiter != nil {
		if reservation := q.limiter.Reserve(); reservation.OK() {
			if delay := reservation.Delay(); delay > backoff {
				return delay
			}
		}
	}
	return backoff
}
