package fan

import "sync"

type Fan[I, O any] struct {
	wg      sync.WaitGroup
	results []O
	errors  []error
}

func New[I, O any]() *Fan[I, O] {
	return &Fan[I, O]{
		wg: sync.WaitGroup{},
	}
}

func (f *Fan[I, O]) Out(s []I, fn func(I) (O, error)) *Fan[I, O] {
	l := len(s)
	if l == 0 {
		return f
	}
	f.results = make([]O, l)
	f.errors = make([]error, l)
	for index, item := range s {
		f.wg.Add(1)
		go func(index int, item I) {
			defer f.wg.Done()
			o, e := fn(item)
			f.results[index] = o
			f.errors[index] = e
		}(index, item)
	}
	return f
}

func (f *Fan[I, O]) In() ([]O, []error, error) {
	f.wg.Wait()
	return f.results, f.errors, nil
}
