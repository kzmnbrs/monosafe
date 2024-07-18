// Package monosafe provides a couple of single-value in-memory
// caches with auto- and manual validation controls.
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

// RevalidateFunc defines cache revalidation. Typically, a repository method
// or an API handle.
//
// It may return the old value.
type RevalidateFunc[T any] func(ctx context.Context, oldValue *T) (*T, error)

type (
	RunOption = any

	// WithManualControl serves a manual revalidation control.
	WithManualControl <-chan struct{}

	// WithTick sets revalidation timer interval. Zero means no timer (manual only).
	//
	// Defaults to [DefaultTick].
	// Negative values are considered invalid.
	WithTick time.Duration

	// WithFuncOnError is called on each revalidation failure, except for the first one.
	WithFuncOnError func(error)
)

const DefaultTick = time.Minute
