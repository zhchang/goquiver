package safe

type wrapError struct {
	err error
}

type Tuple2[T1, T2 any] struct {
	V1 T1
	V2 T2
}

type Tuple3[T1, T2, T3 any] struct {
	V1 T1
	V2 T2
	V3 T3
}

type OK wrapError

func (o OK) Unwrap() {
	if o.err != nil {
		panic(wrapError(o))
	}
}

type Result[T any] struct {
	res T
	err error
}

func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(wrapError{r.err})
	}
	return r.res
}

func Wrap(err error) OK {
	return OK{
		err: err,
	}
}

func Wrap1[T any](t T, err error) Result[T] {
	return Result[T]{
		res: t,
		err: err,
	}
}

func Wrap2[T1, T2 any](v1 T1, v2 T2, err error) Result[Tuple2[T1, T2]] {
	return Result[Tuple2[T1, T2]]{
		res: Tuple2[T1, T2]{
			V1: v1,
			V2: v2,
		},
		err: err,
	}
}

func Wrap3[T1, T2, T3 any](v1 T1, v2 T2, v3 T3, err error) Result[Tuple3[T1, T2, T3]] {
	return Result[Tuple3[T1, T2, T3]]{
		res: Tuple3[T1, T2, T3]{
			V1: v1,
			V2: v2,
			V3: v3,
		},
		err: err,
	}
}

func Chain(f func()) (err error) {
	defer Defer(&err)
	f()
	err = nil
	return
}

func Chain1[T1 any](f func() T1) (t T1, err error) {
	defer Defer(&err)
	return f(), nil
}

func Chain2[T1, T2 any](f func() (T1, T2)) (t1 T1, t2 T2, err error) {
	defer Defer(&err)
	t1, t2 = f()
	return t1, t2, nil
}

func Chain3[T1, T2, T3 any](f func() (T1, T2, T3)) (t1 T1, t2 T2, t3 T3, err error) {
	defer Defer(&err)
	t1, t2, t3 = f()
	return t1, t2, t3, nil
}

// call defer Defer() when safe Result are unwrapped in a function.
// It will continue panic if the panic recovered is no safe internal, otherwise it will return an error
func Defer(err *error) {
	if r := recover(); r != nil {
		wrapErr, ok := r.(wrapError)
		if !ok {
			panic(r)
		}
		*err = wrapErr.err
	}
}
