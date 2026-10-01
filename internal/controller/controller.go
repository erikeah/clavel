package controller

import (
	"context"
	"log/slog"
	"sync"
)

// ReconcileFunc processes a single resource key. Returning nil inside the
// controller marks the key as handled successfully; an error re-enqueues the
// key with rate limiting and backoff.
type ReconcileFunc func(ctx context.Context, key string) error

// Controller runs a pool of workers that reconcile keys produced by an
// Informer through a WorkQueue.
type Controller[R any] struct {
	informer  *Informer[R]
	queue     WorkQueue
	reconcile ReconcileFunc
	workers   int
}

func NewController[R any](informer *Informer[R], reconcile ReconcileFunc, workers int) *Controller[R] {
	return &Controller[R]{
		informer:  informer,
		queue:     informer.queue,
		reconcile: reconcile,
		workers:   workers,
	}
}

func (c *Controller[R]) runWorker(ctx context.Context) {
	for {
		key, shutdown := c.queue.Get()
		if shutdown {
			return
		}
		if err := c.reconcile(ctx, key); err != nil {
			slog.Error("reconcile failed, will retry", "key", key, "error", err)
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}
		c.queue.Done(key)
	}
}

// Run blocks until ctx is cancelled, then shuts down the queue and waits for
// in-flight work to drain.
func (c *Controller[R]) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		wg.Go(func() {
			defer wg.Done()
			c.runWorker(ctx)
		})
	}
	<-ctx.Done()
	c.queue.ShutDown()
	wg.Wait()
}
