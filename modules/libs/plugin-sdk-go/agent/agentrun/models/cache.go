// Package models caches the models a program lists.
package models

import (
	"context"
	"sync"
	"time"
)

// modelListLimit is how long a program has to list its models.
const modelListLimit = 15 * time.Second

// FailureBackoff is how long a key that failed to list is left alone.
const FailureBackoff = 30 * time.Second

// Cache caches what a program listed, for as long as the key asked under stays the same. A
// failure caches nothing, and the key that failed is not asked again for a while. The zero value is
// ready to use.
type Cache[M any] struct {
	mu       sync.Mutex
	key      string
	models   []M
	isCached bool
	// failedKey and failedAt are the key the program last failed to list under, and when.
	failedKey string
	failedAt  time.Time
	// Now is the clock the backoff uses. Nil is the system clock.
	Now func() time.Time
}

// getNow is the time by the clock of the cache.
func (c *Cache[M]) getNow() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

// Get returns the models cached under the key, else those list returns now. A list that fails
// returns none; so does a request that ends before it answers.
func (c *Cache[M]) Get(ctx context.Context, key string, list func(context.Context) ([]M, error)) []M {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isCached && c.key == key {
		return c.models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < FailureBackoff {
		return nil
	}
	asking, stop := context.WithTimeout(ctx, modelListLimit)
	defer stop()
	models, err := list(asking)
	if err != nil {
		if ctx.Err() == nil {
			c.failedKey, c.failedAt = key, c.getNow()
		}
		return nil
	}
	c.key, c.models, c.isCached = key, models, true
	c.failedKey = ""
	return models
}
