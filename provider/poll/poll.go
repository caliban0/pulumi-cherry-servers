package poll

import "time"

// Ticker returns a function that makes channels that send on every interval.
func Ticker(i IntervalFunc) func() <-chan time.Time {
	return func() <-chan time.Time {
		return time.Tick(i())
	}
}