package monosafe

import (
	"context"
	"sync"
)

type Transact[T any] struct {
	value *T
	mtx   sync.RWMutex
	runner[T]
}

func NewTransact[T any](loader Loader[T]) (*Transact[T], error) {
	t := Transact[T]{}
	run, err := newRunner[T](loader,
		func() *T {
			t.mtx.RLock()
			value := t.value
			t.mtx.RUnlock()
			return value
		},
		func(x *T) {
			t.mtx.Lock()
			t.value = x
			t.mtx.Unlock()
		},
	)
	if err != nil {
		return nil, err
	}

	t.runner = run
	return &t, nil
}

func MustTransact[T any](loader Loader[T]) *Transact[T] {
	t, err := NewTransact(loader)
	if err != nil {
		panic(err)
	}
	return t
}

func (t *Transact[T]) Run(ctx context.Context, opts ...RunOption) (*Transact[T], error) {
	return t, t.runner.Run(ctx, opts)
}

// View executes a read-only transaction.
func (t *Transact[T]) View(txn func(*T)) {
	t.mtx.RLock()
	txn(t.value)
	t.mtx.RUnlock()
}

// Update executes a read-write transaction.
func (t *Transact[T]) Update(txn func(*T)) {
	t.mtx.Lock()
	txn(t.value)
	t.mtx.Unlock()
}
