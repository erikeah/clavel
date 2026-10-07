package controller

import "sync"

// Informer maintains a cache of resources observed through a watch stream and
// enqueues their keys into a WorkQueue as they change.
type Informer[R any] struct {
	mu      sync.RWMutex
	cache   map[string]R
	queue   WorkQueue
	keyFunc func(R) string
	// onChange is the reaction to a change; nil means the changed key itself
	// is what gets reconciled.
	onChange func(key string, resource R, present bool) []string
}

// InformerOption configures an Informer.
type InformerOption[R any] func(*Informer[R])

// WithOnChange sets how a change is turned into work. The handler may keep
// whatever indexes it needs from the resource, and returns the keys to enqueue;
// once it is set the informer no longer enqueues the changed key itself.
//
// This is what lets one informer feed another: a controller watching resources
// that merely trigger work elsewhere (an upstream Evaluation waking the
// Artifacts that reference it) returns the dependent keys instead of its own.
func WithOnChange[R any](handler func(key string, resource R, present bool) []string) InformerOption[R] {
	return func(in *Informer[R]) { in.onChange = handler }
}

func NewInformer[R any](queue WorkQueue, keyFunc func(R) string, options ...InformerOption[R]) *Informer[R] {
	informer := &Informer[R]{
		cache:   make(map[string]R),
		queue:   queue,
		keyFunc: keyFunc,
	}
	for _, option := range options {
		option(informer)
	}
	return informer
}

// Add caches a resource (create or update) and enqueues it for reconcile.
func (in *Informer[R]) Add(resource R) {
	key := in.keyFunc(resource)
	in.mu.Lock()
	in.cache[key] = resource
	in.mu.Unlock()
	in.enqueue(in.reacted(key, resource, true))
}

// Delete evicts a resource from the cache and enqueues it so reconcile observes
// its absence.
func (in *Informer[R]) Delete(key string) {
	in.mu.Lock()
	delete(in.cache, key)
	in.mu.Unlock()
	var zero R
	in.enqueue(in.reacted(key, zero, false))
}

// Get returns the cached resource for a key.
func (in *Informer[R]) Get(key string) (R, bool) {
	in.mu.RLock()
	defer in.mu.RUnlock()
	resource, ok := in.cache[key]
	return resource, ok
}

// Replace swaps the cache for a freshly read snapshot: everything the snapshot
// does not contain is evicted, which a replay of puts alone can never do. Every
// resource the snapshot contains and every key it evicts is offered to the
// handler, so a resync resyncs through the same reaction as an ordinary change
// instead of trusting a cache built from a partial history.
func (in *Informer[R]) Replace(snapshot []R) {
	in.mu.Lock()
	previous := in.cache
	in.cache = make(map[string]R, len(snapshot))
	for _, resource := range snapshot {
		in.cache[in.keyFunc(resource)] = resource
	}
	evicted := make([]string, 0, len(previous))
	for key := range previous {
		if _, ok := in.cache[key]; !ok {
			evicted = append(evicted, key)
		}
	}
	in.mu.Unlock()

	// Handlers run without the lock: they may read the cache, and they decide
	// which keys to reconcile just as they do for a single change.
	var keys []string
	for _, resource := range snapshot {
		keys = append(keys, in.reacted(in.keyFunc(resource), resource, true)...)
	}
	var zero R
	for _, key := range evicted {
		keys = append(keys, in.reacted(key, zero, false)...)
	}
	in.enqueue(keys)
}

// reacted hands a change to the configured handler, defaulting to the changed
// key itself. Callers hold no lock, because a handler may read the cache.
func (in *Informer[R]) reacted(key string, resource R, present bool) []string {
	if in.onChange != nil {
		return in.onChange(key, resource, present)
	}
	return []string{key}
}

func (in *Informer[R]) enqueue(keys []string) {
	for _, key := range keys {
		in.queue.Add(key)
	}
}

// List returns a snapshot of all cached resources.
func (in *Informer[R]) List() []R {
	in.mu.RLock()
	defer in.mu.RUnlock()
	out := make([]R, 0, len(in.cache))
	for _, resource := range in.cache {
		out = append(out, resource)
	}
	return out
}
