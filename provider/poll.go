package provider

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

type jitterFunc func() time.Duration

// jitterFromInterval returns a pseudo-random duration, from the half-open
// interval [min,max).
func jitterFromInterval(min, max time.Duration) (jitterFunc, error) {
	if min < 0 || max < 0 || min > max {
		return nil, fmt.Errorf(
			"min %v can't be bigger than max %v and neither can be negative", min, max)
	}
	diff := max.Nanoseconds() - min.Nanoseconds()

	return func() time.Duration {
		return min + time.Duration(rand.Int64N(diff))
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

func withJitter(f jitterFunc) pollerOption {
	return func(p *poller) {
		p.jitter = f
	}
}

func withDelay(d delayFunc) pollerOption {
	return func(p *poller) {
		p.delay = d
	}
}

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
