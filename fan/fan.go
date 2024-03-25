package fan

import "sync"

type Fan[T any] struct {
	wg sync.WaitGroup
}

func New[T any]() *Fan[T] {
	return &Fan[T]{
		wg: sync.WaitGroup{},
	}
}

func (f *Fan[T]) SliceOut(s []T, fn func(T)) *Fan[T] {
	for _, item := range s {
		f.wg.Add(1)
		go func(item T) {
			defer f.wg.Done()
			fn(item)
		}(item)
	}
	return f
}

func (f *Fan[T]) MapOut(m map[string]T, fn func(string, T)) *Fan[T] {
	for key, item := range m {
		f.wg.Add(1)
		go func(key string, item T) {
			defer f.wg.Done()
			fn(key, item)
		}(key, item)
	}
	return f
}

func (f *Fan[T]) In() {
	f.wg.Wait()
}
