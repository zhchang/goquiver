package pool

import (
	"context"
	"sync"
)

type Task func()

type Pool struct {
	wg    sync.WaitGroup
	size  int
	count int
	tasks chan Task
	ctx   context.Context
}

type Option func(*Pool)

func WithContext(ctx context.Context) Option {
	return func(p *Pool) {
		if ctx != nil {
			p.ctx = ctx
		} else {
			p.ctx = context.Background()
		}
	}
}

func WithSize(size int) Option {
	return func(p *Pool) {
		p.size = size
	}
}

func New(opts ...Option) *Pool {
	p := &Pool{
		tasks: make(chan Task),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

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
