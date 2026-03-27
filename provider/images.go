package provider

import (
	"sync"

	"github.com/cherryservers/cherrygo/v3"
	"golang.org/x/sync/singleflight"
)

// SingleFlightMemoizer memoizes functions with a single flight wrapper.
// Useful for preventing redundant API calls for stable values.
type SingleFlightMemoizer[V any] struct {
	mu    sync.RWMutex
	g     singleflight.Group
	cache map[string]V
}

// Memoize will wrap fn in a single flight execution and cache it's result for next call.
func (m *SingleFlightMemoizer[V]) Memoize(fn func(string) (V, error)) func(string) (V, error) {
	return func(k string) (V, error) {
		// Check if value already cached.
		m.mu.RLock()
		v, ok := m.cache[k]
		m.mu.RUnlock()
		if ok {
			return v, nil
		}

		// Lookup value, if it's not cached.
		_, err, _ := m.g.Do(k, func() (any, error) {
			var err error

			// Another goroutine might have already cached the value.
			m.mu.RLock()
			v, ok = m.cache[k]
			m.mu.RUnlock()
			if ok {
				return nil, nil
			}

			v, err = fn(k)
			if err != nil {
				return nil, err
			}

			m.mu.Lock()
			m.cache[k] = v
			m.mu.Unlock()

			return nil, nil
		})
		if err != nil {
			return *new(V), err
		}

		m.mu.RLock()
		v = m.cache[k]
		m.mu.RUnlock()
		return v, nil
	}
}

// NewSingleFlightMemoizer creates a memoizer that caches V type values.
func NewSingleFlightMemoizer[V any]() SingleFlightMemoizer[V] {
	return SingleFlightMemoizer[V]{
		cache: make(map[string]V),
	}
}

type Memoizer[V any] interface {
	Memoize(func(string) (V, error)) func(string) (V, error)
}

// CachedImageClient is Cherry Servers image client that caches results.
type CachedImageClient struct {
	get func(string) ([]string, error)
}

func newCachedImageClient(
	memoizer Memoizer[[]string],
	client cherrygo.ImagesService) ImageClient {
	return &CachedImageClient{
		get: memoizer.Memoize(func(plan string) ([]string, error) {
			images, _, err := client.List(plan, nil)

			slugs := make([]string, len(images))
			for i, image := range images {
				slugs[i] = image.Slug
			}
			return slugs, err
		}),
	}
}

// Get gets image slugs based on a plan slug.
func (c *CachedImageClient) Get(plan string) ([]string, error) {
	images, err := c.get(plan)
	if err != nil {
		return []string{}, err
	}
	newImages := make([]string, len(images))
	copy(newImages, images)
	return newImages, nil
}
