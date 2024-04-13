// Package fan provides a generic implementation of the fan-out pattern in Go.
//
// The main type in this package is Fan, which takes an input of type I, processes it concurrently, and produces an output of type O.
// It uses a pool of goroutines to process the input concurrently.
//
// The package provides two options to customize the behavior of the Fan: WithContext and WithConcurrency.
// WithContext allows you to specify a context that the Fan will respect.
// WithConcurrency allows you to specify the number of goroutines that the Fan will use for processing.
//
// This package uses the "github.com/zhchang/goquiver/pool" package for managing the pool of goroutines.
package fan

import (
	"context"

	"github.com/zhchang/goquiver/pool"
)

type Fan[I, O any] struct {
	results     []O
	errors      []error
	pool        *pool.Pool
	ctx         context.Context
	concurrency int
}

type Option[I, O any] func(*Fan[I, O])

func WithContext[I, O any](ctx context.Context) Option[I, O] {
	return func(f *Fan[I, O]) {
		f.ctx = ctx
	}
}

func WithConcurrency[I, O any](cc int) Option[I, O] {
	return func(f *Fan[I, O]) {
		f.concurrency = cc
	}
}

func New[I, O any](opts ...Option[I, O]) *Fan[I, O] {
	r := &Fan[I, O]{}
	for _, opt := range opts {
		opt(r)
	}
	r.pool = pool.New(pool.WithContext(r.ctx), pool.WithSize(r.concurrency))
	return r
}

func (f *Fan[I, O]) Out(s []I, fn func(I) (O, error)) *Fan[I, O] {
	l := len(s)
	if l == 0 {
		return f
	}
	f.results = make([]O, l)
	f.errors = make([]error, l)
	for index, item := range s {
		_index := index
		_item := item
		f.pool.Run(func() {
			o, e := fn(_item)
			f.results[_index] = o
			f.errors[_index] = e
		})
	}
	return f
}

func (f *Fan[I, O]) In() ([]O, []error, error) {
	err := f.pool.UntilFinished()
	return f.results, f.errors, err
}
