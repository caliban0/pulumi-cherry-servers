package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

var (
	ErrInvalidDuration  = errors.New("invalid duration")
	ErrContextCancelled = errors.New("context cancelled")
)

// TickSource returns a channel that sends a timestamp on every tick.
type TickSource func() (<-chan time.Time, error)

// DurationFunc returns a duration.
type DurationFunc func() time.Duration

// NewTickSource creates a TickSource that ticks every f().
func NewTickSource(f DurationFunc) TickSource {
	return func() (<-chan time.Time, error) {
		d := f()
		if d <= 0 {
			return nil, fmt.Errorf(
				"%w: non-positive duration %v", ErrInvalidDuration, d,
			)
		}
		return time.NewTicker(d).C, nil
	}
}

// DurationWithJitter creates a DurationFunc that adds a little
// bit of jitter to the base duration.
func DurationWithJitter(base time.Duration) (DurationFunc, error) {
	if base < 0 {
		return nil, fmt.Errorf("%w: negative base duration %v", ErrInvalidDuration, base)
	}

	const (
		minJitter = time.Millisecond * 5
		maxJitter = time.Second
	)

	diff := maxJitter - minJitter

	return func() time.Duration {
		jitter := minJitter + time.Duration(rand.Int64N(diff.Nanoseconds())) //nolint:gosec // Safe to use
		// pseudo-random here.
		return base + jitter
	}, nil
}

// Until calls f until it returns true or a non-nil error. Will never make
// any calls to f after the context is cancelled.
func Until(ctx context.Context, ts TickSource, f func() (bool, error)) error {
	// Try immediately, so long as the context is not already cancelled.
	if ctx.Err() == nil {
		r, err := f()
		if r || err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
	}

	c, err := ts()
	if err != nil {
		return fmt.Errorf("failed to get tick source: %w", err)
	}

	for {
		select {
		case <-c:
			r, err := f()
			if r || err != nil {
				return err
			}

		case <-ctx.Done():
			return fmt.Errorf("%w: %w", ErrContextCancelled, ctx.Err())
		}
	}
}
