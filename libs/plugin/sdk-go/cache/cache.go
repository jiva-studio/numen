package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
)

// Option configures a Cache.
type Option func(*config)

type config struct {
	failureBackoff time.Duration
	timeout        time.Duration
	isAsync        bool
	clock          Clock
}

// WithFailureBackoff sets how long a failed load for a key backs off before retrying.
func WithFailureBackoff(d time.Duration) Option {
	return func(c *config) { c.failureBackoff = d }
}

// WithTimeout sets the per-load timeout applied to the context passed to a loader.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithAsync enables background loading. An asynchronous cache with a name also keeps its values on
// disk, in the cache directory of the plugin, and serves them from there while it loads them again.
func WithAsync(isAsync bool) Option {
	return func(c *config) { c.isAsync = isAsync }
}

// Clock gets the current time.
type Clock interface {
	Now() time.Time
}

// WithClock sets the clock used for failure backoff.
func WithClock(source Clock) Option {
	return func(c *config) { c.clock = source }
}

type call[V any] struct {
	readyCh chan struct{}
	value   V
	err     error
}

// Cache keeps keyed values of type V in memory and on disk.
type Cache[V any] struct {
	mu             sync.Mutex
	diskMu         sync.Mutex
	name           string
	failureBackoff time.Duration
	timeout        time.Duration
	isAsync        bool
	clock          Clock
	entries        map[string]V
	failed         map[string]bool
	failedAt       map[string]time.Time
	inFlight       map[string]*call[V]
}

func New[V any](name string, opts ...Option) *Cache[V] {
	cfg := config{clock: clock.System{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	// gomutants:disable-next-line reason="the clock is read only when a failure backoff is set, and the one test that passes a nil clock sets none"
	if cfg.clock == nil {
		cfg.clock = clock.System{}
	}
	return &Cache[V]{
		name:           name,
		failureBackoff: cfg.failureBackoff,
		timeout:        cfg.timeout,
		isAsync:        cfg.isAsync,
		clock:          cfg.clock,
		entries:        make(map[string]V),
		failed:         make(map[string]bool),
		failedAt:       make(map[string]time.Time),
		inFlight:       make(map[string]*call[V]),
	}
}

// SetAsync enables or disables background loading.
func (c *Cache[V]) SetAsync(isAsync bool) {
	c.mu.Lock()
	c.isAsync = isAsync
	c.mu.Unlock()
}

// SetClock sets the clock used for failure backoff.
func (c *Cache[V]) SetClock(source Clock) {
	c.mu.Lock()
	c.clock = source
	c.mu.Unlock()
}

// Put stores value under key in memory and on disk when disk persistence is active.
func (c *Cache[V]) Put(key string, value V) {
	c.mu.Lock()
	c.ensureMapsLocked()
	c.entries[key] = value
	delete(c.failed, key)
	delete(c.failedAt, key)
	shouldWrite := c.shouldUseDiskLocked()
	c.mu.Unlock()
	if shouldWrite {
		c.writeDiskEntry(key, value)
	}
}

// Values returns a snapshot of all entries currently in memory.
func (c *Cache[V]) Values() map[string]V {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]V, len(c.entries))
	for k, value := range c.entries {
		out[k] = value
	}
	return out
}

// HasFailed reports whether the last load for key failed and has not succeeded since.
func (c *Cache[V]) HasFailed(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failed[key]
}

// Get returns the cached value for key, loading it with load when absent.
func (c *Cache[V]) Get(ctx context.Context, key string, load func(context.Context) (V, error)) (V, bool, error) {
	c.mu.Lock()
	c.ensureMapsLocked()
	if value, found := c.entries[key]; found {
		c.mu.Unlock()
		return value, true, nil
	}
	if c.isBackingOffLocked(key) {
		c.mu.Unlock()
		var zero V
		return zero, false, nil
	}
	if c.isAsync {
		value, isFound := c.getAsyncLocked(key, load)
		return value, isFound, nil
	}
	return c.getSyncLocked(ctx, key, load)
}

func (c *Cache[V]) getNow() time.Time {
	c.mu.Lock()
	source := c.clock
	c.mu.Unlock()
	if source == nil {
		return clock.System{}.Now()
	}
	return source.Now()
}

func (c *Cache[V]) getNowLocked() time.Time {
	if c.clock == nil {
		return clock.System{}.Now()
	}
	return c.clock.Now()
}

func (c *Cache[V]) ensureMapsLocked() {
	if c.entries == nil {
		c.entries = make(map[string]V)
	}
	if c.failed == nil {
		c.failed = make(map[string]bool)
	}
	if c.failedAt == nil {
		c.failedAt = make(map[string]time.Time)
	}
	if c.inFlight == nil {
		c.inFlight = make(map[string]*call[V])
	}
}

func (c *Cache[V]) isBackingOffLocked(key string) bool {
	if c.failureBackoff <= 0 || !c.failed[key] {
		return false
	}
	return c.getNowLocked().Sub(c.failedAt[key]) < c.failureBackoff
}

func (c *Cache[V]) shouldUseDiskLocked() bool {
	return c.name != "" && c.isAsync
}

func (c *Cache[V]) getAsyncLocked(key string, load func(context.Context) (V, error)) (V, bool) {
	if _, isInFlight := c.inFlight[key]; isInFlight {
		c.mu.Unlock()
		var zero V
		return zero, false
	}
	if c.shouldUseDiskLocked() {
		return c.loadDiskAndRefreshLocked(key, load)
	}
	c.enqueueAsyncLoadLocked(key, load)
	c.mu.Unlock()
	var zero V
	return zero, false
}

func (c *Cache[V]) loadDiskAndRefreshLocked(key string, load func(context.Context) (V, error)) (V, bool) {
	c.mu.Unlock()
	diskVal, ok := c.readDiskEntry(key)
	c.mu.Lock()
	if ok {
		if _, exists := c.entries[key]; !exists {
			c.entries[key] = diskVal
		}
		c.enqueueAsyncLoadLocked(key, load)
		c.mu.Unlock()
		return diskVal, true
	}
	if latest, exists := c.entries[key]; exists {
		c.mu.Unlock()
		return latest, true
	}
	c.enqueueAsyncLoadLocked(key, load)
	c.mu.Unlock()
	var zero V
	return zero, false
}

func (c *Cache[V]) enqueueAsyncLoadLocked(key string, load func(context.Context) (V, error)) {
	if load == nil {
		return
	}
	if _, isInFlight := c.inFlight[key]; isInFlight {
		return
	}
	inflight := &call[V]{readyCh: make(chan struct{})}
	c.inFlight[key] = inflight
	go func() {
		bgCtx := context.Background()
		val, err := c.executeLoad(bgCtx, load)
		// A failed background load is kept as a failure of the key, which HasFailed reports.
		_, _, _ = c.finishLoad(bgCtx, key, inflight, val, err)
	}()
}

func (c *Cache[V]) getSyncLocked(ctx context.Context, key string, load func(context.Context) (V, error)) (V, bool, error) {
	if inflight, ok := c.inFlight[key]; ok {
		c.mu.Unlock()
		return waitInFlight(ctx, inflight)
	}
	inflight := &call[V]{readyCh: make(chan struct{})}
	c.inFlight[key] = inflight
	c.mu.Unlock()
	val, err := c.executeLoad(ctx, load)
	return c.finishLoad(ctx, key, inflight, val, err)
}

func waitInFlight[V any](ctx context.Context, inflight *call[V]) (V, bool, error) {
	select {
	case <-inflight.readyCh:
		return inflight.value, inflight.err == nil, inflight.err
	case <-ctx.Done():
		var zero V
		return zero, false, ctx.Err()
	}
}

func (c *Cache[V]) executeLoad(ctx context.Context, load func(context.Context) (V, error)) (V, error) {
	if load == nil {
		var zero V
		return zero, errors.New("cache loader is nil")
	}
	if c.timeout <= 0 {
		return load(ctx)
	}
	loadCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return load(loadCtx)
}

func (c *Cache[V]) finishLoad(ctx context.Context, key string, inflight *call[V], val V, err error) (V, bool, error) {
	if err == nil {
		// The value is written before it is visible, so a reader that sees it finds it on disk.
		c.mu.Lock()
		shouldWrite := c.shouldUseDiskLocked()
		c.mu.Unlock()
		if shouldWrite {
			c.writeDiskEntry(key, val)
		}
	}
	now := c.getNow()
	c.mu.Lock()
	delete(c.inFlight, key)
	if err != nil {
		if ctx.Err() == nil {
			c.failed[key] = true
			c.failedAt[key] = now
		}
		inflight.err = err
		close(inflight.readyCh)
		c.mu.Unlock()
		var zero V
		return zero, false, err
	}
	c.entries[key] = val
	delete(c.failed, key)
	delete(c.failedAt, key)
	inflight.value = val
	close(inflight.readyCh)
	c.mu.Unlock()
	return val, true, nil
}
