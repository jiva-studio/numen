package cache_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}

func TestSyncGetLoadsOnceAndKeepsTheValue(t *testing.T) {
	var calls int
	c := cache.New[[]string]("models")
	load := func(context.Context) ([]string, error) {
		calls++
		return []string{"m1", "m2"}, nil
	}

	got1, found1, err1 := c.Get(t.Context(), "k1", load)
	if err1 != nil || !found1 || !slices.Equal(got1, []string{"m1", "m2"}) || calls != 1 {
		t.Fatalf("first Get = %v, %v, %v (calls=%d)", got1, found1, err1, calls)
	}

	got2, found2, err2 := c.Get(t.Context(), "k1", load)
	if err2 != nil || !found2 || !slices.Equal(got2, []string{"m1", "m2"}) || calls != 1 {
		t.Fatalf("second Get = %v, %v, %v (calls=%d)", got2, found2, err2, calls)
	}
}

func TestSyncFailureBackoffSkipsReloadUntilBackoffElapses(t *testing.T) {
	now := newFakeClock(time.Unix(2000, 0))
	var calls int
	c := cache.New[string](
		"models",
		cache.WithFailureBackoff(30*time.Second),
		cache.WithClock(now),
	)
	load := func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", errors.New("boom")
		}
		return "ok", nil
	}

	_, found, err := c.Get(t.Context(), "k1", load)
	if err == nil || found || !c.HasFailed("k1") || calls != 1 {
		t.Fatalf("first Get = found=%v, err=%v, failed=%v, calls=%d", found, err, c.HasFailed("k1"), calls)
	}

	now.Advance(10 * time.Second)
	_, found, err = c.Get(t.Context(), "k1", load)
	if err != nil || found || calls != 1 {
		t.Fatalf("second Get during backoff = found=%v, err=%v, calls=%d", found, err, calls)
	}

	now.Advance(25 * time.Second)
	val, found, err := c.Get(t.Context(), "k1", load)
	if err != nil || !found || val != "ok" || c.HasFailed("k1") || calls != 2 {
		t.Fatalf("third Get after backoff = %q, found=%v, err=%v, failed=%v, calls=%d", val, found, err, c.HasFailed("k1"), calls)
	}
}

func TestAsyncGetLoadsInTheBackground(t *testing.T) {
	plugintest.GrantStorage(t)
	c := cache.New[[]string]("models", cache.WithAsync(true))

	gate := make(chan struct{})
	val, found, err := c.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		<-gate
		return []string{"v1"}, nil
	})
	if err != nil || found || len(val) != 0 || c.HasFailed("k1") {
		t.Fatalf("initial async Get = %v, found=%v, err=%v, failed=%v", val, found, err, c.HasFailed("k1"))
	}

	close(gate)
	waitForCondition(t, func() bool {
		v, ok, err := c.Get(t.Context(), "k1", nil)
		if err != nil {
			t.Fatal(err)
		}
		return ok && slices.Equal(v, []string{"v1"})
	})
}

func TestAsyncGetRecordsFailureAfterBackgroundError(t *testing.T) {
	plugintest.GrantStorage(t)

	c := cache.New[[]string]("models", cache.WithAsync(true))
	gate := make(chan struct{})

	val, found, err := c.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		<-gate
		return nil, errors.New("load failed")
	})
	if err != nil || found || len(val) != 0 || c.HasFailed("k1") {
		t.Fatalf("in-flight Get = %v, found=%v, err=%v, failed=%v", val, found, err, c.HasFailed("k1"))
	}

	close(gate)
	waitForCondition(t, func() bool { return c.HasFailed("k1") })
}

func TestDiskPersistenceAcrossTwoCacheInstances(t *testing.T) {
	plugintest.GrantStorage(t)
	c1 := cache.New[[]string]("models", cache.WithAsync(true))
	if _, _, err := c1.Get(t.Context(), "agent-a", func(context.Context) ([]string, error) {
		return []string{"alpha", "beta"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	waitForCondition(t, func() bool {
		v, ok, err := c1.Get(t.Context(), "agent-a", nil)
		return err == nil && ok && slices.Equal(v, []string{"alpha", "beta"})
	})

	c2 := cache.New[[]string]("models", cache.WithAsync(true))
	refreshGate := make(chan struct{})
	refreshed := make(chan struct{})
	got2, found2, err2 := c2.Get(t.Context(), "agent-a", func(context.Context) ([]string, error) {
		<-refreshGate
		defer close(refreshed)
		return []string{"gamma"}, nil
	})
	if err2 != nil || !found2 || !slices.Equal(got2, []string{"alpha", "beta"}) {
		t.Fatalf("c2.Get from disk = %v, %v, %v", got2, found2, err2)
	}
	close(refreshGate)
	<-refreshed
	waitForCondition(t, func() bool {
		return slices.Equal(c2.Values()["agent-a"], []string{"gamma"})
	})
}

func TestSingleflightDeduplicatesConcurrentCallers(t *testing.T) {
	c := cache.New[string]("models", cache.WithTimeout(5*time.Second))
	var loadCount atomic.Int32
	gate := make(chan struct{})

	const callers = 8
	var wg sync.WaitGroup
	wg.Add(callers)
	results := make([]string, callers)
	errs := make([]error, callers)

	for i := range callers {
		go func(idx int) {
			defer wg.Done()
			val, _, err := c.Get(t.Context(), "shared", func(context.Context) (string, error) {
				loadCount.Add(1)
				<-gate
				return "single-result", nil
			})
			results[idx] = val
			errs[idx] = err
		}(i)
	}

	waitForCondition(t, func() bool { return loadCount.Load() == 1 })
	close(gate)
	wg.Wait()

	if count := loadCount.Load(); count != 1 {
		t.Fatalf("loadCount = %d, want 1", count)
	}
	for i := range callers {
		if errs[i] != nil || results[i] != "single-result" {
			t.Fatalf("caller %d got (%q, %v)", i, results[i], errs[i])
		}
	}
}

func TestValuesSnapshot(t *testing.T) {
	c := cache.New[[]string]("models")
	c.Put("a", []string{"m1"})
	c.Put("b", []string{"m2"})
	vals := c.Values()
	if len(vals) != 2 || !slices.Equal(vals["a"], []string{"m1"}) || !slices.Equal(vals["b"], []string{"m2"}) {
		t.Fatalf("Values() = %v", vals)
	}
}

func waitForCondition(t *testing.T, check func() bool) {
	t.Helper()
	const attempts = 600
	for range attempts {
		if check() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before deadline")
}

func TestADiskWriteFailureKeepsTheMemoryEntry(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cache.EnvCacheDir, filepath.Join(blocker, "sub"))
	c := cache.New[[]string]("models", cache.WithAsync(true))

	c.Put("k1", []string{"m1"})

	if v, ok, err := c.Get(t.Context(), "k1", nil); err != nil || !ok || !slices.Equal(v, []string{"m1"}) {
		t.Errorf("the memory entry = %v, %v, %v, want it kept", v, ok, err)
	}
}

func TestGetPluginDirIsTheDirectoryTheHostGranted(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)

	got, err := cache.GetPluginDir()

	if err != nil || got != cacheDir {
		t.Errorf("GetPluginDir() = %q, %v, want %q", got, err, cacheDir)
	}
}

func TestGetDataDirIsTheDirectoryTheHostGranted(t *testing.T) {
	_, dataDir := plugintest.GrantStorage(t)

	got, err := cache.GetDataDir()

	if err != nil || got != dataDir {
		t.Errorf("GetDataDir() = %q, %v, want %q", got, err, dataDir)
	}
}

func TestAPluginTheHostGrantedNoDirectoryGetsAnError(t *testing.T) {
	t.Setenv(cache.EnvCacheDir, "")
	t.Setenv(cache.EnvDataDir, "")

	if dir, err := cache.GetPluginDir(); err == nil {
		t.Errorf("GetPluginDir() = %q, want an error", dir)
	}
	if dir, err := cache.GetDataDir(); err == nil {
		t.Errorf("GetDataDir() = %q, want an error", dir)
	}
}

func TestTheDiskCacheIsStoredInTheGrantedDirectory(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	first := cache.New[[]string]("models", cache.WithAsync(true))
	first.Put("k", []string{"v"})

	second := cache.New[[]string]("models", cache.WithAsync(true))
	got, found, err := second.Get(t.Context(), "k", nil)

	if err != nil || !found || !slices.Equal(got, []string{"v"}) {
		t.Errorf("Get() = %v, %v, %v, want the entry the first cache wrote", got, found, err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "models.json")); err != nil {
		t.Errorf("the cache file is not in the granted directory: %v", err)
	}
}

func TestASyncCacheKeepsNothingOnDisk(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	c := cache.New[[]string]("models")

	c.Put("k", []string{"v"})

	if entries, err := os.ReadDir(cacheDir); err != nil || len(entries) != 0 {
		t.Errorf("the cache directory holds %v (%v), want nothing", entries, err)
	}
}

func TestGetSharedDirIsTheDirectoryTheHostShares(t *testing.T) {
	shared := plugintest.GrantSharedStorage(t)

	got, err := cache.GetSharedDir()
	own, ownErr := cache.GetSharedDirOrPluginDir()

	if err != nil || got != shared || ownErr != nil || own != shared {
		t.Errorf("GetSharedDir() = %q, %v; GetSharedDirOrPluginDir() = %q, %v; want %q", got, err, own, ownErr, shared)
	}
}

func TestAnOlderHostThatSharesNothingMakesThePluginUseItsOwnDirectory(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	t.Setenv(cache.EnvSharedDir, "")

	if dir, err := cache.GetSharedDir(); err == nil {
		t.Errorf("GetSharedDir() = %q, want an error", dir)
	}
	got, err := cache.GetSharedDirOrPluginDir()

	if err != nil || got != cacheDir {
		t.Errorf("GetSharedDirOrPluginDir() = %q, %v, want %q", got, err, cacheDir)
	}
}

func TestAHostThatGrantedNothingProvidesNoDirectoryAtAll(t *testing.T) {
	t.Setenv(cache.EnvSharedDir, "")
	t.Setenv(cache.EnvCacheDir, "")

	if dir, err := cache.GetSharedDirOrPluginDir(); err == nil {
		t.Errorf("GetSharedDirOrPluginDir() = %q, want an error", dir)
	}
}

func TestGetSharedDataDirIsTheDataDirectoryTheHostShares(t *testing.T) {
	plugintest.GrantStorage(t)
	shared := os.Getenv(cache.EnvSharedDataDir)

	got, err := cache.GetSharedDataDir()
	own, ownErr := cache.GetSharedDataDirOrPluginDataDir()

	if err != nil || got != shared || ownErr != nil || own != shared || shared == os.Getenv(cache.EnvSharedDir) {
		t.Errorf("GetSharedDataDir() = %q, %v; OrPluginDataDir = %q, %v; want %q", got, err, own, ownErr, shared)
	}
}

func TestAnOlderHostThatSharesNoDataMakesThePluginUseItsOwnDataDirectory(t *testing.T) {
	_, dataDir := plugintest.GrantStorage(t)
	t.Setenv(cache.EnvSharedDataDir, "")

	if dir, err := cache.GetSharedDataDir(); err == nil {
		t.Errorf("GetSharedDataDir() = %q, want an error", dir)
	}
	got, err := cache.GetSharedDataDirOrPluginDataDir()

	if err != nil || got != dataDir {
		t.Errorf("GetSharedDataDirOrPluginDataDir() = %q, %v, want %q", got, err, dataDir)
	}
}

func TestCacheZeroValueTimeoutDiskCorruptAndAsyncInFlightEdgeCases(t *testing.T) {
	var zeroCache cache.Cache[string]
	zeroCache.Put("k1", "v1")
	if v, ok, err := zeroCache.Get(t.Context(), "k2", func(context.Context) (string, error) { return "v2", nil }); err != nil || !ok || v != "v2" {
		t.Fatalf("zeroCache.Get = %q, %v, %v", v, ok, err)
	}

	nilClockCache := cache.New[string]("models", cache.WithClock(nil), cache.WithTimeout(50*time.Millisecond))
	var hadDeadline bool
	if _, _, err := nilClockCache.Get(t.Context(), "k", func(ctx context.Context) (string, error) {
		_, hadDeadline = ctx.Deadline()
		return "ok", nil
	}); err != nil || !hadDeadline {
		t.Fatalf("expected deadline from WithTimeout, got err=%v hadDeadline=%v", err, hadDeadline)
	}

	noTimeoutCache := cache.New[string]("models", cache.WithFailureBackoff(time.Second), cache.WithClock(newFakeClock(time.Time{})))
	var noDeadline bool
	if _, _, err := noTimeoutCache.Get(t.Context(), "k", func(ctx context.Context) (string, error) {
		_, noDeadline = ctx.Deadline()
		return "ok", nil
	}); err != nil || noDeadline {
		t.Fatalf("expected no deadline without WithTimeout, got err=%v noDeadline=%v", err, noDeadline)
	}
	if _, _, err := noTimeoutCache.Get(t.Context(), "missing", nil); err == nil {
		t.Fatal("expected error when loader is nil on cache miss")
	}

	dir, _ := plugintest.GrantStorage(t)
	unnamed := cache.New[[]string]("", cache.WithAsync(true))
	unnamed.Put("k", []string{"v"})
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("expected unnamed cache not to write to disk, got %v (%v)", entries, err)
	}

	diskCache := cache.New[[]string]("models", cache.WithAsync(true))
	diskCache.Put("empty", []string{})
	if _, err := os.Stat(filepath.Join(dir, "models.json")); err == nil {
		t.Fatal("expected empty slice not to be persisted to disk")
	}

	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"k1":["corrupt"],"k2":bad}`), 0o600); err != nil {
		t.Fatal(err)
	}
	corruptCache := cache.New[[]string]("models", cache.WithAsync(true))
	if v, found, err := corruptCache.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		return []string{"fresh"}, nil
	}); err != nil || found || len(v) != 0 {
		t.Fatalf("expected corrupt disk JSON to be a miss, got %v, %v, %v", v, found, err)
	}
	waitForCondition(t, func() bool {
		v, ok, err := corruptCache.Get(t.Context(), "k1", nil)
		return err == nil && ok && slices.Equal(v, []string{"fresh"})
	})
	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"k1":["corrupt"],"k2":bad}`), 0o600); err != nil {
		t.Fatal(err)
	}
	corruptCache.Put("k3", []string{"valid"})
	if v, found, err := cache.New[[]string]("models", cache.WithAsync(true)).Get(t.Context(), "k3", nil); err != nil || !found || !slices.Equal(v, []string{"valid"}) {
		t.Fatalf("expected a Put over a corrupt file to write a valid one, got %v, %v, %v", v, found, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"empty":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyReadCache := cache.New[[]string]("models", cache.WithAsync(true))
	if v, found, err := emptyReadCache.Get(t.Context(), "empty", func(context.Context) ([]string, error) {
		return []string{"fresh-after-empty"}, nil
	}); err != nil || found || len(v) != 0 {
		t.Fatalf("expected empty slice on disk to be treated as miss, got %v, %v, %v", v, found, err)
	}
	waitForCondition(t, func() bool {
		v, ok, err := emptyReadCache.Get(t.Context(), "empty", nil)
		return err == nil && ok && slices.Equal(v, []string{"fresh-after-empty"})
	})

	var asyncCalls atomic.Int32
	gate := make(chan struct{})
	started := make(chan struct{})
	diskAsync := cache.New[[]string]("inflight", cache.WithAsync(true))
	_, found, err := diskAsync.Get(t.Context(), "miss", func(context.Context) ([]string, error) {
		if asyncCalls.Add(1) == 1 {
			close(started)
		}
		<-gate
		return []string{"loaded"}, nil
	})
	if err != nil || found {
		t.Fatalf("first async miss = found=%v, err=%v", found, err)
	}
	<-started
	if err := os.WriteFile(filepath.Join(dir, "inflight.json"), []byte(`{"miss":["from-disk"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, found2, err2 := diskAsync.Get(t.Context(), "miss", func(context.Context) ([]string, error) {
		asyncCalls.Add(1)
		return []string{"second"}, nil
	}); err2 != nil || found2 {
		t.Fatalf("second async miss while in-flight = found=%v, err=%v", found2, err2)
	}
	close(gate)
	waitForCondition(t, func() bool {
		v, ok, err := diskAsync.Get(t.Context(), "miss", nil)
		return err == nil && ok && slices.Equal(v, []string{"loaded"})
	})
	if asyncCalls.Load() != 1 {
		t.Fatalf("expected 1 async load call, got %d", asyncCalls.Load())
	}

	var zeroFailCache cache.Cache[string]
	if _, _, err := zeroFailCache.Get(t.Context(), "fail", func(context.Context) (string, error) {
		return "", errors.New("fail")
	}); err == nil || !zeroFailCache.HasFailed("fail") {
		t.Fatalf("expected zeroFailCache error and HasFailed=true, got %v", err)
	}
}
