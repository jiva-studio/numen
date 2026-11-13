package rowoptions

import (
	"maps"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// CacheTTL is how long a listing of options is answered from the cache.
const CacheTTL = 10 * time.Minute

// Cache keeps the options the providers listed, by connection, row field, row options and operations, for
// CacheTTL. A listing counts only for the connection as it was read then: a change of its revision, its
// fields or the version of its plugin makes the next listing ask the provider. Failures are not kept.
type Cache struct {
	clock   machineport.Clock
	mu      sync.Mutex
	entries map[cacheKey]cacheEntry
}

// NewCache returns an empty cache that times its entries by clock.
func NewCache(clock machineport.Clock) *Cache {
	return &Cache{clock: clock, entries: map[cacheKey]cacheEntry{}}
}

type cacheKey struct {
	connectionID string
	field        string
	options      string
	operations   string
}

type cacheEntry struct {
	state    connectionState
	listedAt time.Time
	options  []modelcall.Option
}

// connectionState is what of a connection a listing of options depends on.
type connectionState struct {
	revision      int
	pluginVersion string
	fields        map[string]string
}

func newCacheKey(connectionID, field string, options map[string]string, operations []string) cacheKey {
	pairs := make([]string, 0, len(options))
	for _, key := range slices.Sorted(maps.Keys(options)) {
		pairs = append(pairs, key+"\x00"+options[key])
	}
	return cacheKey{
		connectionID: connectionID, field: field, options: strings.Join(pairs, "\x01"),
		operations: strings.Join(operations, "\x01"),
	}
}

func newConnectionState(resolved connection.Resolved) connectionState {
	return connectionState{revision: resolved.Revision, pluginVersion: resolved.PluginVersion, fields: resolved.Fields}
}

func (s connectionState) isEqual(other connectionState) bool {
	return s.revision == other.revision && s.pluginVersion == other.pluginVersion && maps.Equal(s.fields, other.fields)
}

func (c *Cache) get(key cacheKey, state connectionState) ([]modelcall.Option, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, isFound := c.entries[key]
	if !isFound || !entry.state.isEqual(state) || c.clock.Now().Sub(entry.listedAt) >= CacheTTL {
		return nil, false
	}
	return slices.Clone(entry.options), true
}

// put keeps the listing and drops the listings of the connection as it was before a change.
func (c *Cache) put(key cacheKey, state connectionState, options []modelcall.Option) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for other, entry := range c.entries {
		if other.connectionID == key.connectionID && !entry.state.isEqual(state) {
			delete(c.entries, other)
		}
	}
	c.entries[key] = cacheEntry{state: state, listedAt: c.clock.Now(), options: slices.Clone(options)}
}
