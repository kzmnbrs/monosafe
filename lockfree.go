package monosafe

import (
	"context"
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

func MustLockFree[T any](reval RevalidateFunc[T]) *LockFree[T] {
	f, err := NewLockFree(reval)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *LockFree[T]) Run(ctx context.Context, opts ...RunOption) (*LockFree[T], error) {
	return f, f.runner.Run(ctx, opts)
}
