package provider_test

import (
	"testing"

	"github.com/caliban0/pulumi-cherry-servers/provider"
	"github.com/stretchr/testify/assert"
)

func TestHappyMemoize(t *testing.T) {
	m := provider.NewSingleFlightMemoizer[string]()
	calls := 0
	var f = func(string) (string, error) {
		calls++
		return "happy", nil
	}
	memoized := m.Memoize(f)

	for range 1000 {
		go func() {
			got, err := memoized("test")
			assert.Equal(t, "happy", got)
			assert.NoError(t, err)
		}()
	}

	assert.Equal(t, 1, calls)
}
