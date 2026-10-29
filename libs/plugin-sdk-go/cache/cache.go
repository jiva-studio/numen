package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache/internal/adapter/clock"
)

// Option configures a Cache.
type Option func(*config)

type config struct {
	pluginID       string
	dir            string
	ttl            time.Duration
	failureBackoff time.Duration
	timeout        time.Duration
	isAsync        bool
	isDiskEnabled  bool
	now            func() time.Time
}

// WithPluginID sets the plugin identifier used to locate the disk cache directory.
func WithPluginID(id string) Option {
	return func(c *config) { c.pluginID = id }
}

// WithDir sets an explicit directory for disk cache files.
func WithDir(dir string) Option {
	return func(c *config) { c.dir = dir }
}

// WithTTL sets how long an in-memory entry remains fresh before a refresh is triggered.
func WithTTL(d time.Duration) Option {
	return func(c *config) { c.ttl = d }
}

// WithFailureBackoff sets how long a failed load for a key backs off before retrying.
func WithFailureBackoff(d time.Duration) Option {
	return func(c *config) { c.failureBackoff = d }
}

// WithTimeout sets the per-load timeout applied to the context passed to a loader.
func WithTimeout(d time.Duration) Option {
	return func(c *config) { c.timeout = d }
}

// WithAsync enables stale-while-revalidate background loading.
func WithAsync(isAsync bool) Option {
	return func(c *config) { c.isAsync = isAsync }
}

// WithDisk enables disk persistence even when asynchronous mode is off.
func WithDisk(isEnabled bool) Option {
	return func(c *config) { c.isDiskEnabled = isEnabled }
}

// WithClock sets the clock used for expiration and failure backoff.
func WithClock(now func() time.Time) Option {
	return func(c *config) { c.now = now }
}

type entry[V any] struct {
	value     V
	fetchedAt time.Time
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
	pluginID       string
	dir            string
	ttl            time.Duration
	failureBackoff time.Duration
	timeout        time.Duration
	isAsync        bool
	isDiskEnabled  bool
	now            func() time.Time
	entries        map[string]entry[V]
	failed         map[string]bool
	failedAt       map[string]time.Time
	inFlight       map[string]*call[V]
}

// New creates a Cache named name with the given options.
func New[V any](name string, opts ...Option) *Cache[V] {
	var cfg config
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Cache[V]{
		name:           name,
		pluginID:       cfg.pluginID,
		dir:            cfg.dir,
		ttl:            cfg.ttl,
		failureBackoff: cfg.failureBackoff,
		timeout:        cfg.timeout,
		isAsync:        cfg.isAsync,
		isDiskEnabled:  cfg.isDiskEnabled,
		now:            cfg.now,
		entries:        make(map[string]entry[V]),
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

// SetClock sets the clock used for expiration and failure backoff.
func (c *Cache[V]) SetClock(now func() time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

// Put stores value under key in memory and on disk when disk persistence is active.
func (c *Cache[V]) Put(key string, value V) {
	now := c.getNow()
	c.mu.Lock()
	c.ensureMapsLocked()
	c.entries[key] = entry[V]{value: value, fetchedAt: now}
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
	for k, one := range c.entries {
		out[k] = one.value
	}
	return out
}

// HasFailed reports whether the last load for key failed and has not succeeded since.
func (c *Cache[V]) HasFailed(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failed[key]
}

// Get returns the cached value for key, loading it with load when absent or expired.
func (c *Cache[V]) Get(ctx context.Context, key string, load func(context.Context) (V, error)) (V, bool, error) {
	c.mu.Lock()
	c.ensureMapsLocked()
	current, found := c.entries[key]
	if found && c.isFreshLocked(current.fetchedAt) {
		c.mu.Unlock()
		return current.value, true, nil
	}
	if c.isBackingOffLocked(key) {
		c.mu.Unlock()
		return current.value, found, nil
	}
	if c.isAsync {
		return c.getAsyncLocked(key, current.value, found, load)
	}
	return c.getSyncLocked(ctx, key, found, load)
}

func (c *Cache[V]) getNow() time.Time {
	c.mu.Lock()
	now := c.now
	c.mu.Unlock()
	if now == nil {
		return clock.Now()
	}
	return now()
}

func (c *Cache[V]) getNowLocked() time.Time {
	if c.now == nil {
		return clock.Now()
	}
	return c.now()
}

func (c *Cache[V]) ensureMapsLocked() {
	if c.entries == nil {
		c.entries = make(map[string]entry[V])
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

func (c *Cache[V]) isFreshLocked(fetchedAt time.Time) bool {
	if c.ttl <= 0 {
		return true
	}
	return c.getNowLocked().Sub(fetchedAt) < c.ttl
}

func (c *Cache[V]) isBackingOffLocked(key string) bool {
	if c.failureBackoff <= 0 || !c.failed[key] {
		return false
	}
	return c.getNowLocked().Sub(c.failedAt[key]) < c.failureBackoff
}

func (c *Cache[V]) shouldUseDiskLocked() bool {
	if c.name == "" {
		return false
	}
	return c.isAsync || c.isDiskEnabled || c.dir != ""
}

func (c *Cache[V]) getAsyncLocked(key string, current V, found bool, load func(context.Context) (V, error)) (V, bool, error) {
	if found {
		c.enqueueAsyncLoadLocked(key, load)
		c.mu.Unlock()
		return current, true, nil
	}
	if _, isInFlight := c.inFlight[key]; isInFlight {
		c.mu.Unlock()
		var zero V
		return zero, false, nil
	}
	if c.shouldUseDiskLocked() {
		return c.loadDiskAndRefreshLocked(key, load)
	}
	c.enqueueAsyncLoadLocked(key, load)
	c.mu.Unlock()
	var zero V
	return zero, false, nil
}

func (c *Cache[V]) loadDiskAndRefreshLocked(key string, load func(context.Context) (V, error)) (V, bool, error) {
	c.mu.Unlock()
	diskVal, ok := c.readDiskEntry(key)
	now := c.getNow()
	c.mu.Lock()
	if ok {
		if _, exists := c.entries[key]; !exists {
			c.entries[key] = entry[V]{value: diskVal, fetchedAt: now}
		}
		c.enqueueAsyncLoadLocked(key, load)
		c.mu.Unlock()
		return diskVal, true, nil
	}
	if latest, exists := c.entries[key]; exists {
		c.mu.Unlock()
		return latest.value, true, nil
	}
	c.enqueueAsyncLoadLocked(key, load)
	c.mu.Unlock()
	var zero V
	return zero, false, nil
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
		_, _, _ = c.finishLoad(bgCtx, key, inflight, val, err)
	}()
}

func (c *Cache[V]) getSyncLocked(ctx context.Context, key string, found bool, load func(context.Context) (V, error)) (V, bool, error) {
	if inflight, ok := c.inFlight[key]; ok {
		c.mu.Unlock()
		return waitInFlight(ctx, inflight)
	}
	if !found && c.shouldUseDiskLocked() {
		return c.loadDiskOrFetchSyncLocked(ctx, key, load)
	}
	return c.startSyncLoadLocked(ctx, key, load)
}

func (c *Cache[V]) loadDiskOrFetchSyncLocked(ctx context.Context, key string, load func(context.Context) (V, error)) (V, bool, error) {
	c.mu.Unlock()
	if diskVal, ok := c.readDiskEntry(key); ok {
		now := c.getNow()
		c.mu.Lock()
		if _, exists := c.entries[key]; !exists {
			c.entries[key] = entry[V]{value: diskVal, fetchedAt: now}
		}
		c.mu.Unlock()
		return diskVal, true, nil
	}
	c.mu.Lock()
	if current, ok := c.entries[key]; ok && c.isFreshLocked(current.fetchedAt) {
		c.mu.Unlock()
		return current.value, true, nil
	}
	if inflight, ok := c.inFlight[key]; ok {
		c.mu.Unlock()
		return waitInFlight(ctx, inflight)
	}
	return c.startSyncLoadLocked(ctx, key, load)
}

func (c *Cache[V]) startSyncLoadLocked(ctx context.Context, key string, load func(context.Context) (V, error)) (V, bool, error) {
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
	c.entries[key] = entry[V]{value: val, fetchedAt: now}
	delete(c.failed, key)
	delete(c.failedAt, key)
	shouldWrite := c.shouldUseDiskLocked()
	inflight.value = val
	close(inflight.readyCh)
	c.mu.Unlock()
	if shouldWrite {
		c.writeDiskEntry(key, val)
	}
	return val, true, nil
}

func (c *Cache[V]) cachePath() (string, error) {
	if c.name == "" {
		return "", errors.New("cache name is empty")
	}
	if c.dir != "" {
		return filepath.Join(c.dir, c.name+".json"), nil
	}
	pluginID := c.pluginID
	if pluginID == "" {
		pluginID = os.Getenv("NUMEN_PLUGIN_ID")
	}
	if pluginID == "" {
		return "", errors.New("plugin ID is empty")
	}
	base, err := getUserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "numen", "plugins", pluginID, c.name+".json"), nil
}

func getUserCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err == nil && dir != "" {
		return dir, nil
	}
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return "", err
	}
	return filepath.Join(home, ".cache"), nil
}

func (c *Cache[V]) readDiskEntry(key string) (V, bool) {
	var zero V
	path, err := c.cachePath()
	if err != nil {
		return zero, false
	}
	c.diskMu.Lock()
	data, err := os.ReadFile(path)
	c.diskMu.Unlock()
	if err != nil {
		return zero, false
	}
	var all map[string]V
	if json.Unmarshal(data, &all) != nil {
		return zero, false
	}
	val, found := all[key]
	if !found || !isPersistable(val) {
		return zero, false
	}
	return val, true
}

func (c *Cache[V]) writeDiskEntry(key string, val V) {
	if !isPersistable(val) {
		return
	}
	path, err := c.cachePath()
	if err != nil {
		return
	}
	c.diskMu.Lock()
	defer c.diskMu.Unlock()

	all := make(map[string]V)
	if existing, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(existing, &all)
	}
	all[key] = val
	data, err := json.Marshal(all)
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	writeFileAtomic(dir, path, data)
}

func writeFileAtomic(dir, path string, data []byte) {
	tmp, err := os.CreateTemp(dir, ".cache-*.tmp")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		return
	}
	if os.Rename(tmpName, path) != nil {
		_ = os.Remove(tmpName)
	}
}

func isPersistable[V any](val V) bool {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map {
		return rv.Len() > 0
	}
	return !rv.IsZero()
}
