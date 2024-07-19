// Package monosafe provides a couple of single-value in-memory
// caches with auto- and manual reload controls.
//
// Anticipated workloads are read-heavy, with none-to-little writes.
// e.g. caching smaller lookup tables or API responses.
//
// Prefer [LockFree] implementation over [Transact] if you don't
// need consistent views or partial updates.
//
// Usage: initialise the implementation of choice and call [LockFree.Run]/[Transact.Run].
package monosafe

import (
	"context"
	"time"
)

type (
	// Loader defines cache reload. Typically, a repository method
	// or an API handle.
	//
	// It may return the old value.
	Loader[T any] interface {
		Load(ctx context.Context, oldValue *T) (*T, error)
	}

	LoaderFunc[T any] func(ctx context.Context, oldValue *T) (*T, error)
)

type (
	RunOption = any

	// WithManualControl serves a manual reload control.
	WithManualControl <-chan struct{}

	// WithTick sets reload timer interval. Zero means no timer (manual only).
	//
	// Defaults to [DefaultTick].
	// Negative values are considered invalid.
	WithTick time.Duration

	// WithFuncOnError is called on each reload failure, except for the first one.
	WithFuncOnError func(error)
)

const DefaultTick = time.Minute

func (f LoaderFunc[T]) Load(ctx context.Context, oldValue *T) (*T, error) {
	return f(ctx, oldValue)
}
