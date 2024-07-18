package monosafe

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SpyImpl struct {
	revalNum atomic.Int32
	value    atomic.Pointer[int]
	runner[int]
}

func TestNewRunner(t *testing.T) {
	reval := func(ctx context.Context, oldValue *int) (*int, error) {
		return nil, nil
	}
	load := func() *int { return nil }
	swap := func(*int) {}

	t.Run("ok", func(t *testing.T) {
		run, err := newRunner(reval, load, swap)
		assert.NotZero(t, run)
		assert.Nil(t, err)
	})

	t.Run("no revalidate func", func(t *testing.T) {
		run, err := newRunner[int](nil, load, swap)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})

	t.Run("no load func", func(t *testing.T) {
		run, err := newRunner[int](reval, nil, swap)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})

	t.Run("no swap func", func(t *testing.T) {
		run, err := newRunner[int](reval, load, nil)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})
}

func TestRunner_Run(t *testing.T) {
	spyImpl := func() *SpyImpl {
		impl := SpyImpl{}
		run, _ := newRunner[int](
			func(ctx context.Context, oldValue *int) (*int, error) {
				impl.revalNum.Add(1)
				num := rand.Intn(42)
				return &num, nil
			},
			func() *int {
				return impl.value.Load()
			},
			func(newValue *int) {
				impl.value.Store(newValue)
			},
		)
		impl.runner = run
		return &impl
	}

	t.Run("default", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithTick(time.Millisecond*10),
		))
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.GreaterOrEqual(t, int32(5), impl.revalNum.Load())
	})

	t.Run("no timer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualControl(mc),
			WithTick(0),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.Equal(t, int32(3), impl.revalNum.Load())
	})

	t.Run("negative tick", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		impl := spyImpl()
		assert.Error(t, impl.Run(ctx,
			WithTick(-1),
		))
	})

	t.Run("no triggers", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		impl := spyImpl()
		assert.Error(t, impl.Run(ctx,
			WithManualControl(nil),
			WithTick(0),
		))
	})

	t.Run("manual control closure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualControl(mc),
			WithTick(0),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		close(mc)

		// Checking if chan close doesn't lead to a live lock.
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.Equal(t, int32(3), impl.revalNum.Load())
	})

	t.Run("manual control closure with timer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualControl(mc),
			WithTick(time.Millisecond*10),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		time.Sleep(time.Millisecond * 40) // 5
		close(mc)

		time.Sleep(time.Millisecond * 40) // 4
		cancel()

		assert.True(t, impl.revalNum.Load() >= 9 && impl.revalNum.Load() < 12)
	})
}
