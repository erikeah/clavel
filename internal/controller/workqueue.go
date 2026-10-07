package controller

import (
	"container/heap"
	"math/rand/v2"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	defaultRetryBaseDelay = time.Second
	defaultRetryMaxDelay  = 30 * time.Minute
)

// WorkQueue is a rate-limited, deduplicating queue of string keys, mirroring
// the Kubernetes client-go workqueue semantics.
type WorkQueue interface {
	// Add enqueues a key. A key already waiting in the queue is not enqueued twice.
	Add(key string)
	// AddAfter enqueues a key after delay. A key already waiting to be added
	// again is not rescheduled; the earliest pending delay wins.
	AddAfter(key string, delay time.Duration)
	// AddRateLimited enqueues a key after a per-key retry backoff and an optional
	// global rate limiter delay. Requeues a key that is currently processing;
	// delivery happens once the delay elapses or Done runs, whichever is later.
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
	dirty      map[string]bool
	processing map[string]bool
	retries    map[string]int

	// waitingForAdd tracks keys with a pending delayed add so each key has at
	// most one entry in delayed.
	waitingForAdd map[string]bool
	delayed       delayedHeap
	// wakeLoop tells waitingLoop to recompute its deadline whenever the delayed
	// heap head may have moved. Buffered, so a signal never blocks the sender
	// and one pending signal is enough to catch a head change.
	wakeLoop chan struct{}

	limiter   *rate.Limiter
	baseDelay time.Duration
	maxDelay  time.Duration

	shutDown bool
	stopCh   chan struct{}
}

func NewWorkQueue(options ...WorkQueueOption) WorkQueue {
	q := &workQueue{
		dirty:         make(map[string]bool),
		processing:    make(map[string]bool),
		retries:       make(map[string]int),
		waitingForAdd: make(map[string]bool),
		baseDelay:     defaultRetryBaseDelay,
		maxDelay:      defaultRetryMaxDelay,
		wakeLoop:      make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	for _, option := range options {
		option(q)
	}
	go q.waitingLoop()
	return q
}

func (q *workQueue) Add(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.add(key)
}

// add marks a key dirty and enqueues it unless it is already queued or being
// processed; Done re-enqueues keys re-added while processing. Callers hold q.mu.
func (q *workQueue) add(key string) {
	if q.shutDown {
		return
	}
	if q.dirty[key] {
		return
	}
	q.dirty[key] = true
	if q.processing[key] {
		return
	}
	q.ready = append(q.ready, key)
	q.cond.Signal()
}

func (q *workQueue) AddAfter(key string, delay time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.shutDown {
		return
	}
	q.addAfter(key, delay)
}

func (q *workQueue) AddRateLimited(key string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.shutDown {
		return
	}
	q.retries[key]++
	q.addAfter(key, q.backoffDelay(key))
}

// addAfter schedules a delayed add. Callers hold q.mu.
func (q *workQueue) addAfter(key string, delay time.Duration) {
	if delay <= 0 {
		q.add(key)
		return
	}
	if q.waitingForAdd[key] {
		return
	}
	q.waitingForAdd[key] = true
	heap.Push(&q.delayed, delayedItem{key: key, readyAt: time.Now().Add(delay)})
	// The heap head may have moved in either direction, so waitingLoop has to
	// wake up and recompute how long it must sleep.
	select {
	case q.wakeLoop <- struct{}{}:
	default:
	}
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
		if q.shutDown {
			return "", true
		}
		// Waiting on the condition variable rather than on a timer of our own
		// is what makes Add wake us: a signal delivered while this goroutine
		// slept on a timer would be lost, and a ready key would then wait out
		// an unrelated key's backoff. waitingLoop promotes due delayed items
		// and broadcasts, so every path back to a ready key reaches us.
		q.cond.Wait()
	}
}

// waitingLoop promotes delayed items once their delay elapses. It owns that
// promotion so Get can wait purely on the condition variable, and it is the
// only thing that can wake itself early: a new delayed item may be due sooner
// than the one currently heading the heap.
func (q *workQueue) waitingLoop() {
	for {
		q.mu.Lock()
		if q.shutDown {
			q.mu.Unlock()
			return
		}
		if len(q.delayed) > 0 && !q.delayed[0].readyAt.After(time.Now()) {
			q.promoteDue()
			q.cond.Broadcast()
			q.mu.Unlock()
			continue
		}
		var deadline time.Time
		if len(q.delayed) > 0 {
			deadline = q.delayed[0].readyAt
		}
		q.mu.Unlock()

		var timeout <-chan time.Time
		var timer *time.Timer
		if !deadline.IsZero() {
			timer = time.NewTimer(time.Until(deadline))
			timeout = timer.C
		}
		select {
		case <-q.stopCh:
			if timer != nil {
				timer.Stop()
			}
			return
		case <-q.wakeLoop:
			if timer != nil {
				timer.Stop()
			}
		case <-timeout:
		}
	}
}

// promoteDue moves every delayed item whose delay elapsed into the ready queue.
// Callers hold q.mu.
func (q *workQueue) promoteDue() {
	now := time.Now()
	for len(q.delayed) > 0 && !q.delayed[0].readyAt.After(now) {
		item := heap.Pop(&q.delayed).(delayedItem)
		delete(q.waitingForAdd, item.key)
		q.add(item.key)
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
	if q.shutDown {
		q.mu.Unlock()
		return
	}
	q.shutDown = true
	close(q.stopCh)
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *workQueue) ShuttingDown() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.shutDown
}

// backoffDelay computes the retry delay for a key: exponential backoff based on
// the requeue count with additive jitter (uniform in [base, 2*base), clamped to
// the cap), bounded by the global rate limiter when configured.
func (q *workQueue) backoffDelay(key string) time.Duration {
	backoff := q.baseDelay
	for i := 1; i < q.retries[key]; i++ {
		backoff *= 2
		if backoff >= q.maxDelay {
			backoff = q.maxDelay
			break
		}
	}
	if backoff < q.maxDelay {
		backoff += time.Duration(rand.Int64N(int64(backoff)))
		if backoff > q.maxDelay {
			backoff = q.maxDelay
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
