package provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickSourceErrorWithNonPositiveDuration(t *testing.T) {
	cases := []struct {
		name     string
		duration func() time.Duration
	}{
		{
			name: "zero-duration",
			duration: func() time.Duration {
				return 0
			},
		},
		{
			name: "negative-duration",
			duration: func() time.Duration {
				return -1
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ts := provider.NewTickSource(tt.duration)

			_, err := ts()
			assert.ErrorIs(t, err, provider.ErrInvalidDuration)
		})
	}
}

func FuzzDurationWithJitter(f *testing.F) {
	// Fuzzing only works with primitives, so we need to convert.
	testcases := []int64{
		int64((-10 * time.Second)),
		-1,
		0,
		1,
		int64((10 * time.Second)),
	}
	for _, tc := range testcases {
		f.Add(tc)
	}

	const (
		expectedMinJitter = 5 * time.Millisecond
		expectedMaxJitter = time.Second
	)

	f.Fuzz(func(t *testing.T, a int64) {
		d := time.Duration(a)

		durFunc, err := provider.DurationWithJitter(d)

		// Expect error with negative base duration.
		if d < 0 {
			assert.ErrorIs(t, err, provider.ErrInvalidDuration)
			return
		}

		require.NoError(t, err)

		got := durFunc()
		assert.GreaterOrEqual(t, got, d+expectedMinJitter)
		assert.Less(t, got, d+expectedMaxJitter)
	})
}

func newFakeTickSource(c chan time.Time) provider.TickSource {
	return func() (<-chan time.Time, error) {
		return c, nil
	}
}

func TestUntilWithPreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var (
		tsChan chan time.Time
		calls  = 0
	)

	fts := newFakeTickSource(tsChan)

	err := provider.Until(ctx, fts, func() (bool, error) {
		calls++
		return true, nil
	})

	assert.Zero(t, calls)
	assert.ErrorIs(t, err, provider.ErrContextCancelled)
}

func TestUntilWithImmediateReturnTrue(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var (
		tsChan chan time.Time
		calls  = 0
	)

	fts := newFakeTickSource(tsChan)

	err := provider.Until(ctx, fts, func() (bool, error) {
		calls++
		return true, nil
	})

	assert.Equal(t, 1, calls)
	assert.NoError(t, err)
}

func TestUntilWhenReturnsTrueBeforeContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	const expectedCalls = 5

	var (
		tsChan = make(chan time.Time)
		calls  = 0
	)

	fts := newFakeTickSource(tsChan)

	go func() {
		for range expectedCalls {
			tsChan <- time.Now()
			time.Sleep(time.Millisecond)
		}
	}()

	err := provider.Until(ctx, fts, func() (bool, error) {
		if calls >= expectedCalls {
			return true, nil
		}
		calls++
		return false, nil
	})

	assert.Equal(t, expectedCalls, calls)
	assert.NoError(t, err)
}

func TestUntilWhenContextCancelledBeforeReturnsTrue(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	const ticks = 5

	var (
		tsChan = make(chan time.Time)
		calls  = 0
	)

	fts := newFakeTickSource(tsChan)

	go func() {
		for i := range ticks {
			// Cancel midway.
			if i >= ticks/2 {
				cancel()
			}
			tsChan <- time.Now()
			time.Sleep(time.Millisecond)
		}
	}()

	err := provider.Until(ctx, fts, func() (bool, error) {
		if calls >= ticks {
			return true, nil
		}
		calls++
		return false, nil
	})

	assert.Less(t, calls, ticks)
	assert.ErrorIs(t, err, provider.ErrContextCancelled)
}
