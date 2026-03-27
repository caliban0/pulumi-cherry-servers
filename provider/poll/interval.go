package poll

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// IntervalFunc creates polling time intervals.
type IntervalFunc func () time.Duration

// FixedIntervalFunc returns a function that generates intervals from a fixed base
// with some random jitter from the half-open interval [minJit,maxJit).
func FixedIntervalFunc(minJit, maxJit, base time.Duration) (IntervalFunc, error){
	if base <= 0 {
		return nil, fmt.Errorf("non-positive base duration %q", base)
	}

	diff := maxJit - minJit
	if diff <= 0 {
		return nil, fmt.Errorf("maxJitter %q must be more than minJitter %q", maxJit, minJit)
	}

	return func() time.Duration {
		jitter := minJit + time.Duration(rand.Int64N(diff.Nanoseconds())) //nolint:gosec // Crypto safety no needed here.
		return base + jitter
	}, nil
}