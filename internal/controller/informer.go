package controller

import "sync"

// Informer maintains a cache of resources observed through a watch stream and
// enqueues their keys into a WorkQueue as they change.
type Informer[R any] struct {
	mu      sync.RWMutex
	cache   map[string]R
	queue   WorkQueue
	keyFunc func(R) string
}

func NewInformer[R any](queue WorkQueue, keyFunc func(R) string) *Informer[R] {
	return &Informer[R]{
		cache:   make(map[string]R),
		queue:   queue,
		keyFunc: keyFunc,
	}
}

// Add caches a resource (create or update) and enqueues its key for reconcile.
func (in *Informer[R]) Add(resource R) {
	key := in.keyFunc(resource)
	in.mu.Lock()
	in.cache[key] = resource
	in.mu.Unlock()
	in.queue.Add(key)
}

// Delete evicts a resource from the cache and enqueues its key so reconcile
// observes its absence.
func (in *Informer[R]) Delete(key string) {
	in.mu.Lock()
	delete(in.cache, key)
	in.mu.Unlock()
	in.queue.Add(key)
}

// Get returns the cached resource for a key.
func (in *Informer[R]) Get(key string) (R, bool) {
	in.mu.RLock()
	defer in.mu.RUnlock()
	resource, ok := in.cache[key]
	return resource, ok
}

// Replace swaps the cache for a freshly read snapshot: everything the snapshot
// does not contain is evicted, which a replay of puts alone can never do. Both
// the snapshot keys and the previously cached keys are enqueued, so reconcile
// resyncs on reconnect instead of trusting a cache built from a partial
// history.
func (in *Informer[R]) Replace(snapshot []R) {
	in.mu.Lock()
	previous := in.cache
	in.cache = make(map[string]R, len(snapshot))
	keys := make(map[string]struct{}, len(snapshot))
	for _, resource := range snapshot {
		key := in.keyFunc(resource)
		in.cache[key] = resource
		keys[key] = struct{}{}
	}
	for key := range previous {
		keys[key] = struct{}{}
	}
	in.mu.Unlock()
	for key := range keys {
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
