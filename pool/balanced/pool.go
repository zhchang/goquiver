// Package balanced provides a BalancedPool utility, which allows for managing a pool of workers to execute tasks concurrently.
// "Balanced" because it will distribute the work load (calculated using Task.ETA()) evenly (best effort) among workers.
//
// The pool supports the following features:
//
// - Task scheduling with an ETA (Estimated Time of Arrival) for each task.
// - Error handling for common issues such as workers not found, tasks taking longer than expected, and tasks not being processed.
//
// Here is a basic example of how to use the pool:
//
//	p := pool.New(2)
//	task := pool.TaskImpl{
//	    Do: func() {
//	        // Task logic here
//	    },
//	    ETA: func() time.Duration {
//	        // Return estimated time of arrival
//	        return time.Second * 5
//	    },
//	}
//	p.Schedule(&task)
//
// Note: This package uses the GoLLRB package for maintaining a balanced binary search tree of tasks, and the logrus package for logging.
package balanced

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	btree "github.com/petar/GoLLRB/llrb"
	"github.com/sirupsen/logrus"
	"github.com/zhchang/goquiver/safe"
)

// ErrWorkerNotFound is an error indicating that a worker was not found in the pool.
var ErrWorkerNotFound = fmt.Errorf("unexpected error: worker not found")

// ErrWillTakeLonger is an error indicating that a task is expected to take longer than the specified maximum wait time.
var ErrWillTakeLonger = fmt.Errorf("we are bound to take need longer than maxWait specified")

// ErrUnprocessed is an error indicating that a task was not processed.
var ErrUnprocessed = fmt.Errorf("task not processed")

// Task is an interface representing a task that can be executed by a worker.
type Task interface {
	Do()
	ETA() time.Duration
}

// taskImpl is an implementation of the Task interface.
type taskImpl struct {
	do  func()
	eta func() time.Duration
}

// Do executes the task.
func (t *taskImpl) Do() {
	t.do()
}

// ETA returns the estimated time of arrival for the task.
func (t *taskImpl) ETA() time.Duration {
	return t.eta()
}

// NewTask creates a new Task with the given do function and eta function.
func NewTask(do func(), eta func() time.Duration) Task {
	return &taskImpl{do: do, eta: eta}
}

// taskWrapper is a wrapper struct that holds a task and a channel to signal when the task is finished.
type taskWrapper struct {
	task     Task
	finished chan error
}

// worker represents a worker that executes tasks.
type worker struct {
	sync.RWMutex
	eta     time.Duration
	tasks   *safe.UnlimitedChannel[*taskWrapper]
	wg      *sync.WaitGroup
	ctx     context.Context
	ongoing struct {
		start time.Time
		eta   time.Duration
	}
}

// newWorker creates a new worker with the given wait group and context.
func newWorker(wg *sync.WaitGroup, ctx context.Context) *worker {
	w := &worker{
		tasks: safe.NewUnlimitedChannel[*taskWrapper](),
		wg:    wg,
		ctx:   ctx,
	}
	w.wg.Add(1)
	go w.run()
	return w
}

// ETA returns the estimated time of arrival for the worker.
func (w *worker) getEta() time.Duration {
	w.RLock()
	defer w.RUnlock()
	return w.eta
}

func (w *worker) etaWithOnGoing() time.Duration {
	w.RLock()
	defer w.RUnlock()
	eta := w.eta
	if !w.ongoing.start.IsZero() {
		eta -= min(w.ongoing.eta, time.Since(w.ongoing.start))
	}
	return eta
}

// addTask adds a task to the worker's task queue and returns a channel to signal when the task is finished.
func (w *worker) addTask(task Task) <-chan error {
	finished := make(chan error)
	teta := task.ETA()
	func() {
		w.Lock()
		defer w.Unlock()
		w.eta += teta
	}()
	w.tasks.In() <- &taskWrapper{task: task, finished: finished}
	return finished
}

// Less compares the worker with another worker based on their ETA values.
func (w *worker) Less(item btree.Item) bool {
	w1, ok := item.(*worker)
	if !ok {
		return false
	}
	return w.getEta() < w1.getEta()
}

// run starts the worker's execution loop.
func (w *worker) run() {
	defer func() {
		w.wg.Done()
	}()
	handleTw := func(tw *taskWrapper) {
		if tw == nil {
			//tasks channel closed
			return
		}
		func() {
			w.Lock()
			defer w.Unlock()
			w.ongoing.start = time.Now()
			w.ongoing.eta = tw.task.ETA()
		}()
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("[Task Do Panic]: %s", string(debug.Stack()))
				}
			}()
			tw.task.Do()
		}()
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("[Task ETA Panic]: %s", string(debug.Stack()))
				}
			}()
			w.Lock()
			defer w.Unlock()
			w.eta -= tw.task.ETA()
			w.ongoing.start = time.Time{}
			w.ongoing.eta = 0
		}()
		close(tw.finished)
	}
outer:
	for {
		select {
		case <-w.ctx.Done():
			break outer
		default:
			select {
			case <-w.ctx.Done():
				break outer
			case tw := <-w.tasks.Out():
				handleTw(tw)
			}
		}
	}

	tws := w.tasks.Finalize()
	for _, tw := range tws {
		tw.finished <- ErrUnprocessed
		close(tw.finished)
	}

}

// Pool represents a pool of workers that can execute tasks concurrently.
type Pool struct {
	sync.RWMutex
	wg      sync.WaitGroup
	workers *btree.LLRB
	ctx     context.Context
}

// PoolOption is a function that configures the Pool.
type PoolOption func(*Pool)

// WithContext sets the context for the BalancedPool.
func WithContext(ctx context.Context) PoolOption {
	return func(p *Pool) {
		if ctx != nil {
			p.ctx = ctx
		}
	}
}

// New creates a new BalancedPool with the specified number of workers and options.
func New(workerCount int, opts ...PoolOption) *Pool {
	p := &Pool{
		workers: btree.New(),
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.ctx == nil {
		p.ctx = context.Background()
	}
	for i := 0; i < workerCount; i++ {
		w := newWorker(&p.wg, p.ctx)
		w.eta = time.Duration(i) * time.Microsecond
		_ = p.workers.ReplaceOrInsert(w)
	}
	return p
}

// runOptions represents the options for running a task.
type runOptions struct {
	maxDelay time.Duration
}

// RunOption is a function that configures the options for running a task.
type RunOption func(*runOptions)

// WithMaxDelay sets the maximum delay for running a task.
func WithMaxDelay(d time.Duration) RunOption {
	return func(r *runOptions) {
		r.maxDelay = d
	}
}

// RunAsync runs the given task asynchronously in the balanced pool.
// It accepts an optional variadic parameter `options` of type `RunOption`.
// The `RunOption` is a function that can be used to modify the behavior of the task execution.
// The function returns a channel of type `<-chan error` that can be used to receive any errors that occur during task execution,
// and an error value that indicates if there was an error while running the task.
// If the minimum worker in the pool is not found, it returns `ErrWorkerNotFound`.
// If the maximum delay specified in the options is exceeded by the worker's estimated time of arrival (ETA),
// it returns `ErrWillTakeLonger`.
// The `finished` channel returned can be used to wait for the task to complete.
func (p *Pool) RunAsync(task Task, options ...RunOption) (<-chan error, error) {
	p.Lock()
	defer p.Unlock()
	var ropts runOptions
	for _, option := range options {
		option(&ropts)
	}
	min := p.workers.Min()
	if min == nil {
		return nil, ErrWorkerNotFound
	}
	worker := min.(*worker)
	if ropts.maxDelay != 0 {
		if worker.etaWithOnGoing() > ropts.maxDelay {
			return nil, ErrWillTakeLonger
		}
	}
	finished := worker.addTask(task)
	p.workers.DeleteMin()
	p.workers.InsertNoReplace(worker)
	return finished, nil
}

// Run executes the given task synchronously on the BalancedPool.
// It returns an error if the task execution fails.
func (p *Pool) Run(task Task, options ...RunOption) error {
	var finished <-chan error
	var err error
	if finished, err = p.RunAsync(task, options...); err != nil {
		return err
	}
	return <-finished
}
