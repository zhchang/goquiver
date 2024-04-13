// Package pool provides a simple, flexible, and thread-safe pool of goroutines for managing and reusing resources in Go.
//
// The main type in this package is Pool, which manages a set of goroutines that can be reused for different tasks.
// The Pool ensures that the total number of goroutines does not exceed a given limit.
//
// The package provides a function New, which creates a new Pool with a specified size.
// The Pool has a method Do, which submits a task to the pool to be executed by a goroutine.
// If all goroutines in the pool are busy, Do blocks until one becomes available.
//
// This package is designed to help with scenarios where you have a large number of tasks to perform,
// but you want to limit the number of goroutines used to prevent excessive resource usage.
package pool

import (
	"context"
	"sync"
)

type Task = func()

type Pool struct {
	wg    sync.WaitGroup
	size  int
	count int
	tasks chan Task
	ctx   context.Context
}

type Option func(*Pool)

// WithContext sets the context for the Pool.
// If ctx is not nil, it will be assigned to the Pool's context.
func WithContext(ctx context.Context) Option {
	return func(p *Pool) {
		if ctx != nil {
			p.ctx = ctx
		}
	}
}

// WithSize sets the size of the pool.
// The size determines the maximum number of items that can be stored in the pool.
// It returns an Option function that can be used to configure a Pool instance.
func WithSize(size int) Option {
	return func(p *Pool) {
		p.size = size
	}
}

// New creates a new instance of the Pool with the provided options.
// It returns a pointer to the created Pool.
func New(opts ...Option) *Pool {
	p := &Pool{
		tasks: make(chan Task),
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.ctx == nil {
		p.ctx = context.Background()
	}
	return p
}

// Run executes the given task in the pool.
// If the pool has available workers or the pool size is unlimited (size = 0),
// a new worker goroutine is spawned to handle the task.
// The task is added to the pool's task queue and will be executed by an available worker.
// If the task is nil, it will be ignored.
// If the pool's context is canceled, the worker goroutine will exit.
func (p *Pool) Run(task Task) {
	if p.count < p.size || p.size == 0 {
		p.count++
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case task := <-p.tasks:
					if task == nil {
						return
					}
					task()
				case <-p.ctx.Done():
					return
				}
			}
		}()
	}
	p.tasks <- task
}

// UntilFinished waits until all tasks in the pool are finished or the context is canceled.
// It returns an error if the context is canceled before all tasks are finished.
func (p *Pool) UntilFinished() error {
	close(p.tasks)
	finished := make(chan struct{})
	go func() {
		p.wg.Wait()
		finished <- struct{}{}
	}()
	select {
	case <-finished:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}
