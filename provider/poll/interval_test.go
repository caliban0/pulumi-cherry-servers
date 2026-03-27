package poll_test

import (
	"testing"
	"time"

	"github.com/caliban0/pulumi-cherry-servers/provider/poll"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixedIntervalFunc(t *testing.T) {
	cases := []struct {
		name string
		wantErr bool
		minJit time.Duration
		maxJit time.Duration
		base time.Duration
	} {
		{
		name: "negative base duration",
		wantErr: true,
		minJit: time.Second,
		maxJit: time.Second * 2,
		base: -1,
		},
		{
		name: "zero base duration",
		wantErr: true,
		minJit: time.Second,
		maxJit: time.Second * 2,
		base: 0,
		},
		{
		name: "minJit equals maxJit",
		wantErr: true,
		minJit: time.Second,
		maxJit: time.Second,
		base: 1,
		},
		{
		name: "minJit more than maxJit",
		wantErr: true,
		minJit: time.Second * 2,
		maxJit: time.Second,
		base: 1,
		},
		{
		name: "up to one second jitter",
		wantErr: false,
		minJit: time.Second,
		maxJit: time.Second * 2,
		base: time.Second,
		},
	}

	const sampleCount = 10

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f, err := poll.FixedIntervalFunc(tt.minJit, tt.maxJit, tt.base)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			unique := make(map[time.Duration]struct{}, sampleCount)
			for range sampleCount {
				sample := f()

				assert.GreaterOrEqual(t, sample, tt.minJit + tt.base)
				assert.Less(t, sample, tt.maxJit + tt.base)

				unique[sample] = struct{}{}
			}

			assert.Greater(t, len(unique), 1, "expected at least two distinct durations")
		})
	}
}