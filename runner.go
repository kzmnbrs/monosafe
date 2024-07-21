package monosafe

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

type runner[T any] struct {
	guard  *atomic.Bool
	loader Loader[T]
	get    getFunc[T]
	swap   swapFunc[T]
}

type (
	getFunc[T any]  func() *T
	swapFunc[T any] func(newValue *T)
)

func newRunner[T any](loader Loader[T], get getFunc[T], swap swapFunc[T]) (runner[T], error) {
	if loader == nil {
		return runner[T]{}, errors.New("no loader")
	}
	if get == nil {
		return runner[T]{}, errors.New("no get func")
	}
	if swap == nil {
		return runner[T]{}, errors.New("no swap func")
	}
	return runner[T]{
		guard:  &atomic.Bool{},
		loader: loader,
		get:    get,
		swap:   swap,
	}, nil
}

// Run starts reload timer, which is reset by either [WithManualReload]
// or [DefaultReloadInterval] ([WithReloadTimer]).
//
// Returns initial load error. Consecutive errors can be observed [WithFuncOnError].
//
// Panics when called more than once.
func (r *runner[T]) Run(ctx context.Context, opts ...RunOption) error {
	if r.guard.Swap(true) {
		panic("double run")
	}

	var (
		manualReload <-chan struct{}
		tick         = DefaultReloadInterval
	)
	for i := range opts {
		switch opt := opts[i].(type) {
		case WithManualReload:
			manualReload = opt
		case WithReloadTimer:
			tick = time.Duration(opt)
		}
	}
	if tick < 0 {
		return fmt.Errorf("negative tick value: %v", tick)
	}
	if manualReload == nil && tick == 0 {
		return errors.New("either reload signal or tick must be set")
	}

	newValue, err := r.loader.Load(ctx, r.get())
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
			case _, more := <-manualReload:
				if !more {
					if timer != nil {
						manualReload = nil
						continue
					}
					return // No reload triggers left.
				}
			case <-timerReval:
			}
			if timer != nil {
				timer.Reset(tick)
			}

			oldValue := r.get()
			newValue, err := r.loader.Load(ctx, oldValue)
			if err != nil {
				continue
			}

			if newValue != oldValue {
				r.swap(newValue)
			}
		}
	}()

	return nil
}

// Get retrieves the value.
func (r *runner[T]) Get() *T {
	return r.get()
}

// Swap the value. Thread-safe.
func (r *runner[T]) Swap(newValue *T) {
	r.swap(newValue)
}
