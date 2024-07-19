package monosafe

import (
	"context"
	"sync/atomic"
)

type LockFree[T any] struct {
	runner[T]
	value atomic.Pointer[T]
}

func NewLockFree[T any](loader Loader[T]) (*LockFree[T], error) {
	f := LockFree[T]{}
	run, err := newRunner[T](loader,
		func() *T { return f.value.Load() },
		func(x *T) { f.value.Store(x) },
	)
	if err != nil {
		return nil, err
	}

	f.runner = run
	return &f, nil
}

func MustLockFree[T any](loader Loader[T]) *LockFree[T] {
	f, err := NewLockFree(loader)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *LockFree[T]) Run(ctx context.Context, opts ...RunOption) (*LockFree[T], error) {
	return f, f.runner.Run(ctx, opts)
}
