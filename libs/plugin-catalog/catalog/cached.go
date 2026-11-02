package catalog

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// CachedSource answers from the snapshot of a source while it is fresh, and from the last snapshot
// when the source cannot be read.
type CachedSource struct {
	source Source
	cache  Cache
	ttl    time.Duration
	now    func() time.Time
}

// NewCachedSource returns the source read through the cache. A snapshot whose age is below ttl is
// not read again; now is the clock the age is taken from.
func NewCachedSource(source Source, cache Cache, ttl time.Duration, now func() time.Time) *CachedSource {
	return &CachedSource{source: source, cache: cache, ttl: ttl, now: now}
}

// Name is the name of the source behind the cache.
func (c *CachedSource) Name() string { return c.source.Name() }

// List returns the fresh snapshot, else the entries of the source, which become the snapshot. When
// the source fails, the last snapshot is returned with the failure; with none, the failure alone.
func (c *CachedSource) List(ctx context.Context) ([]Entry, error) {
	snapshot, readErr := c.cache.Read(ctx)
	hasSnapshot := readErr == nil
	if hasSnapshot && c.now().Sub(snapshot.At) < c.ttl {
		return snapshot.Entries, nil
	}
	var problems []error
	if !hasSnapshot && !errors.Is(readErr, ErrEmptyCache) {
		problems = append(problems, fmt.Errorf("read the catalog cache: %w", readErr))
	}
	entries, err := c.source.List(ctx)
	if err != nil {
		problems = append(problems, err)
		if hasSnapshot {
			return snapshot.Entries, errors.Join(problems...)
		}
		return entries, errors.Join(problems...)
	}
	if writeErr := c.cache.Write(ctx, Snapshot{Entries: entries, At: c.now()}); writeErr != nil {
		problems = append(problems, fmt.Errorf("write the catalog cache: %w", writeErr))
	}
	return entries, errors.Join(problems...)
}

// Get returns the entry of the plugin from the listing, which is read as List reads it.
func (c *CachedSource) Get(ctx context.Context, id string) (Entry, error) {
	entries, err := c.List(ctx)
	for _, entry := range entries {
		if entry.ID == id {
			return entry, err
		}
	}
	return Entry{}, errors.Join(fmt.Errorf("plugin %q: %w", id, ErrNotFound), err)
}
