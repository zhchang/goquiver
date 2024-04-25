package safe

import (
	"context"
)

type UnlimitedChannel[T any] struct {
	in        chan T
	out       chan T
	storage   []T
	cancel    func()
	finalized chan struct{}
	closed    bool
}

func NewUnlimitedChannel[T any]() *UnlimitedChannel[T] {
	r := &UnlimitedChannel[T]{
		in:        make(chan T),
		out:       make(chan T),
		storage:   []T{},
		finalized: make(chan struct{}),
	}
	var ctx context.Context
	ctx, r.cancel = context.WithCancel(context.Background())
	go r.manager(ctx, r.finalized)
	return r
}

func (c *UnlimitedChannel[T]) manager(ctx context.Context, finalized chan struct{}) {
	takeIn := func(t T, ok bool) {
		if !ok {
			close(c.out)
			return
		}
		c.storage = append(c.storage, t)
	}
normal:
	for {
		if len(c.storage) == 0 {
			// If storage is empty, wait for new item
			t, ok := <-c.in
			takeIn(t, ok)
		}
		select {
		case t, ok := <-c.in:
			takeIn(t, ok)
		case c.out <- c.storage[0]:
			c.storage = c.storage[1:]
		case <-ctx.Done():
			close(c.out)
			break normal
		}
	}
	close(finalized)
	//safety:
	for {
		<-c.in
	}
}

func (c *UnlimitedChannel[T]) In() chan T {
	return c.in
}

func (c *UnlimitedChannel[T]) Out() <-chan T {
	return c.out
}

func (c *UnlimitedChannel[T]) Finalize() []T {
	c.cancel()
	//wait for finalize to happen
	<-c.finalized
	return c.storage
}
