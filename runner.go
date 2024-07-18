package monosafe

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type runner[T any] struct {
	guard *atomic.Bool
	reval RevalidateFunc[T]
	load  loadFunc[T]
	swap  swapFunc[T]
}

type (
	loadFunc[T any] func() *T
	swapFunc[T any] func(newValue *T)
)

func newRunner[T any](reval RevalidateFunc[T], load loadFunc[T], swap swapFunc[T]) (runner[T], error) {
	if reval == nil {
		return runner[T]{}, errors.New("no revalidation func")
	}
	if load == nil {
		return runner[T]{}, errors.New("no load func")
	}
	if swap == nil {
		return runner[T]{}, errors.New("no swap func")
	}
	return runner[T]{
		guard: &atomic.Bool{},
		reval: reval,
		load:  load,
		swap:  swap,
	}, nil
}

// Run starts revalidation timer, which is reset by either [WithManualControl]
// or [DefaultTick] ([WithTick]).
//
// Returns initial revalidation error. Consecutive errors can be observed [WithFuncOnError].
//
// Panics when called more than once.
func (r *runner[T]) Run(ctx context.Context, opts ...RunOption) error {
	if r.guard.Swap(true) {
		panic("double run")
	}

	var (
		manualReval <-chan struct{}
		tick        = DefaultTick
		onError     = func(error) {}
	)
	for i := range opts {
		switch opt := opts[i].(type) {
		case WithManualControl:
			manualReval = opt
		case WithTick:
			tick = time.Duration(opt)
		case WithFuncOnError:
			onError = opt
		}
	}
	if tick < 0 {
		return fmt.Errorf("negative tick value: %v", tick)
	}
	if manualReval == nil && tick == 0 {
		return errors.New("either revalidation signal or tick must be set")
	}

	newValue, err := r.reval(ctx, r.load())
	if err != nil {
		return err
	}
	r.swap(newValue)

	go func() {
		var (
			timer      *time.Timer
			timerReval <-chan time.Time
		)
		if tick > 0 {
			timer = time.NewTimer(tick)
			defer timer.Stop()

			timerReval = timer.C
		}

		for {
			select {
			case <-ctx.Done():
				return
			case _, more := <-manualReval:
				if !more {
					if timer != nil {
						manualReval = nil
						continue
					}
					return // No revalidation triggers left.
				}
			case <-timerReval:
			}
			if timer != nil {
				timer.Reset(tick)
			}

			oldValue := r.Load()
			newValue, err := r.reval(ctx, oldValue)
			if err != nil {
				onError(err)
				continue
			}

			if newValue != oldValue {
				r.swap(newValue)
			}
		}
	}()

	return nil
}

// Load retrieves the value.
func (r *runner[T]) Load() *T {
	return r.load()
}

// Swap the value. Thread-safe.
func (r *runner[T]) Swap(newValue *T) {
	r.swap(newValue)
}
