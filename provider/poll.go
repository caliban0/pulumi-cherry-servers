package provider

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

type jitterFunc func() time.Duration

// jitterFromInterval returns a pseudo-random duration, from the half-open
// interval [least,most).
func jitterFromInterval(least, most time.Duration) (jitterFunc, error) {
	if least < 0 || most < 0 || least > most {
		return nil, fmt.Errorf(
			"least %v can't be bigger than most %v and neither can be negative", least, most)
	}
	diff := most.Nanoseconds() - least.Nanoseconds()

	return func() time.Duration {
		return least + time.Duration(rand.Int64N(diff)) //nolint:gosec // Safe to use
		// pseudo-random here.
	}, nil
}

type delayFunc func() time.Duration

func constantDelay(delay time.Duration) delayFunc {
	return func() time.Duration {
		return delay
	}
}

type poller struct {
	jitter jitterFunc
	delay  delayFunc
}

// until polls until f returns true or a non-nil error.
func (p poller) until(ctx context.Context, f func(ctx context.Context) (bool, error)) error {
	// Try immediately, so long as the context is not already cancelled.
	if ctx.Err() == nil {
		r, err := f(ctx)
		if r || err != nil {
			return err
		}
	} else {
		return fmt.Errorf(
			"context cancelled prior to condition being fulfilled: %w", ctx.Err())
	}

	timer := time.NewTimer(p.delay() + p.jitter())

	for {
		select {
		case <-timer.C:
			r, err := f(ctx)
			if r || err != nil {
				return err
			}
			timer.Reset(p.delay() + p.jitter())

		case <-ctx.Done():
			return fmt.Errorf(
				"context cancelled prior to condition being fulfilled: %w", ctx.Err())
		}
	}
}

type pollerOption func(p *poller)

func newPoller(opts ...pollerOption) poller {
	const (
		minJitter = time.Second * 1
		maxJitter = time.Second * 2
		delay     = time.Second * 10
	)

	jitter, err := jitterFromInterval(minJitter, maxJitter)
	if err != nil {
		panic(fmt.Sprintf("failed to build jitter func: %v", err))
	}

	p := poller{
		jitter: jitter,
		delay:  constantDelay(delay),
	}

	for _, opt := range opts {
		opt(&p)
	}

	return p
}
