package monosafe

import (
	"sync/atomic"
)

type LockFree[T any] struct {
	runner[T]
	value atomic.Pointer[T]
}

func NewLockFree[T any](reval RevalidateFunc[T]) (*LockFree[T], error) {
	f := LockFree[T]{}
	run, err := newRunner[T](reval,
		func() *T { return f.value.Load() },
		func(x *T) { f.value.Store(x) },
	)
	if err != nil {
		return nil, err
	}

	f.runner = run
	return &f, nil
}

func MustLockFree[T any](loadFunc RevalidateFunc[T]) *LockFree[T] {
	lf, err := NewLockFree(loadFunc)
	if err != nil {
		panic(err)
	}
	return lf
}
