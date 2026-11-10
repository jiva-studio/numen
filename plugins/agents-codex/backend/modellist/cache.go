// Package modellist lists the models Codex offers, and remembers the list while the command line
// that produced it stays the same.
package modellist

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
)

// listLimit is how long Codex has to list its models.
const listLimit = 15 * time.Second

// failureBackoff is how long a command line that failed to list is left alone.
const failureBackoff = 30 * time.Second

// Source is where the models come from.
type Source interface {
	// List asks the program of command for its models.
	List(ctx context.Context, command []string) ([]agentmodel.Model, error)
	// ReadStored returns the list the program kept from an earlier run, or none.
	ReadStored() []agentmodel.Model
}

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// GetModels returns the models the source lists for command, or the stored list when the source
// lists none.
func GetModels(ctx context.Context, source Source, command []string) ([]agentmodel.Model, error) {
	models, err := source.List(ctx, command)
	if len(models) > 0 {
		return models, nil
	}
	if stored := source.ReadStored(); len(stored) > 0 {
		return stored, nil
	}
	return nil, err
}

// Cache caches what the source listed, for as long as the command line stays the same. A failure
// caches nothing, and the command line that failed is not retried for a while. An asynchronous cache
// answers with the stored list at once and lists in the background.
type Cache struct {
	mu        sync.Mutex
	key       string
	models    []agentmodel.Model
	isCached  bool
	failedKey string
	failedAt  time.Time
	inFlight  map[string]bool
	isAsync   bool
	clock     Clock
	source    Source
}

// NewCache creates an empty cache over source.
func NewCache(source Source, clock Clock, isAsync bool) *Cache {
	return &Cache{
		mu: sync.Mutex{}, key: "", models: nil, isCached: false, failedKey: "", failedAt: time.Time{},
		inFlight: nil, isAsync: isAsync, clock: clock, source: source,
	}
}

func (c *Cache) getNow() time.Time {
	return c.clock.Now()
}

// Get returns the models of the program that command starts.
func (c *Cache) Get(ctx context.Context, command []string) []agentmodel.Model {
	key := strings.Join(command, " ")
	c.mu.Lock()
	if c.isCached && c.key == key {
		models := c.models
		c.mu.Unlock()
		return models
	}
	if c.failedKey == key && c.getNow().Sub(c.failedAt) < failureBackoff {
		c.mu.Unlock()
		return nil
	}
	if c.isAsync {
		c.startBackgroundFetch(ctx, key, command)
		c.mu.Unlock()
		return c.source.ReadStored()
	}
	c.mu.Unlock()
	return c.fetchAndStore(ctx, key, command)
}

func (c *Cache) startBackgroundFetch(ctx context.Context, key string, command []string) {
	if c.inFlight[key] {
		return
	}
	if c.inFlight == nil {
		c.inFlight = make(map[string]bool)
	}
	c.inFlight[key] = true
	go c.fetchInBackground(context.WithoutCancel(ctx), key, slices.Clone(command))
}

func (c *Cache) fetchInBackground(ctx context.Context, key string, command []string) {
	defer func() {
		c.mu.Lock()
		delete(c.inFlight, key)
		c.mu.Unlock()
	}()
	c.fetchAndStore(ctx, key, command)
}

func (c *Cache) fetchAndStore(ctx context.Context, key string, command []string) []agentmodel.Model {
	asking, stop := context.WithTimeout(ctx, listLimit)
	defer stop()
	models, err := GetModels(asking, c.source, command)
	c.mu.Lock()
	defer c.mu.Unlock()
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
