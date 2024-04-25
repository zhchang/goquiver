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
}

// NewUnlimitedChannel creates a new instance of UnlimitedChannel[T].
// It initializes the necessary channels and storage for the channel.
// It also starts a goroutine to manage the channel's operations.
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
			select {
			case t, ok := <-c.in:
				takeIn(t, ok)
			case <-ctx.Done():
				close(c.out)
				break normal
			}
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

// In returns the input channel of the UnlimitedChannel.
func (c *UnlimitedChannel[T]) In() chan T {
	return c.in
}

// Out returns a receive-only channel of type T. This channel can be used to receive values sent to the UnlimitedChannel.
func (c *UnlimitedChannel[T]) Out() <-chan T {
	return c.out
}

// Finalize closes the channel and waits for the finalization to happen.
// It returns the storage of the channel.
// Note: Write to In() won't block even after finalize, but it won't be effective also
func (c *UnlimitedChannel[T]) Finalize() []T {
	c.cancel()
	//wait for finalize to happen
	<-c.finalized
	return c.storage
}
