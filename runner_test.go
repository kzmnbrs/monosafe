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
	reloadNum atomic.Int32
	value     atomic.Pointer[int]
	runner[int]
}

func TestNewRunner(t *testing.T) {
	loader := LoaderFunc[int](func(ctx context.Context, oldValue *int) (*int, error) {
		return nil, nil
	})
	get := func() *int { return nil }
	swap := func(*int) {}

	t.Run("ok", func(t *testing.T) {
		run, err := newRunner(loader, get, swap)
		assert.NotZero(t, run)
		assert.Nil(t, err)
	})

	t.Run("no loader", func(t *testing.T) {
		run, err := newRunner[int](nil, get, swap)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})

	t.Run("no get func", func(t *testing.T) {
		run, err := newRunner[int](loader, nil, swap)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})

	t.Run("no swap func", func(t *testing.T) {
		run, err := newRunner[int](loader, get, nil)
		assert.Zero(t, run)
		assert.NotNil(t, err)
	})
}

func TestRunner_Run(t *testing.T) {
	spyImpl := func() *SpyImpl {
		impl := SpyImpl{}
		run, _ := newRunner[int](
			LoaderFunc[int](func(ctx context.Context, oldValue *int) (*int, error) {
				impl.reloadNum.Add(1)
				num := rand.Intn(42)
				return &num, nil
			}),
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
			WithReloadTimer(time.Millisecond*10),
		))
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.GreaterOrEqual(t, int32(5), impl.reloadNum.Load())
	})

	t.Run("no timer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualReload(mc),
			WithReloadTimer(0),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.Equal(t, int32(3), impl.reloadNum.Load())
	})

	t.Run("negative tick", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		impl := spyImpl()
		assert.Error(t, impl.Run(ctx,
			WithReloadTimer(-1),
		))
	})

	t.Run("no triggers", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		impl := spyImpl()
		assert.Error(t, impl.Run(ctx,
			WithManualReload(nil),
			WithReloadTimer(0),
		))
	})

	t.Run("manual control closure", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualReload(mc),
			WithReloadTimer(0),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		close(mc)

		// Checking if chan close doesn't lead to a live lock.
		time.Sleep(time.Millisecond * 40)
		cancel()

		assert.Equal(t, int32(3), impl.reloadNum.Load())
	})

	t.Run("manual control closure with timer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mc := make(chan struct{})

		impl := spyImpl()
		assert.NoError(t, impl.Run(ctx,
			WithManualReload(mc),
			WithReloadTimer(time.Millisecond*10),
		))
		mc <- struct{}{}
		mc <- struct{}{}
		time.Sleep(time.Millisecond * 40) // 5
		close(mc)

		time.Sleep(time.Millisecond * 40) // 4
		cancel()

		assert.True(t, impl.reloadNum.Load() >= 9 && impl.reloadNum.Load() < 12)
	})
}
