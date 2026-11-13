package cache_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/plugintest"
)

func TestSyncGetCachesHitsAndExpiresAfterTTL(t *testing.T) {
	now := newFakeClock(time.Unix(1000, 0))
	var calls int
	c := cache.New[[]string](
		"models",
		cache.WithTTL(10*time.Minute),
		cache.WithClock(now),
	)
	load := func(context.Context) ([]string, error) {
		calls++
		return []string{"m1", "m2"}, nil
	}

	got1, found1, err1 := c.Get(t.Context(), "k1", load)
	if err1 != nil || !found1 || !slices.Equal(got1, []string{"m1", "m2"}) || calls != 1 {
		t.Fatalf("first Get = %v, %v, %v (calls=%d)", got1, found1, err1, calls)
	}

	now.Advance(5 * time.Minute)
	got2, found2, err2 := c.Get(t.Context(), "k1", load)
	if err2 != nil || !found2 || !slices.Equal(got2, []string{"m1", "m2"}) || calls != 1 {
		t.Fatalf("second Get within TTL = %v, %v, %v (calls=%d)", got2, found2, err2, calls)
	}

	now.Advance(6 * time.Minute)
	got3, found3, err3 := c.Get(t.Context(), "k1", load)
	if err3 != nil || !found3 || !slices.Equal(got3, []string{"m1", "m2"}) || calls != 2 {
		t.Fatalf("third Get after TTL = %v, %v, %v (calls=%d)", got3, found3, err3, calls)
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

func TestAsyncGet_ServesStaleWhileRevalidating(t *testing.T) {
	plugintest.GrantStorage(t)

	now := newFakeClock(time.Unix(3000, 0))

	c := cache.New[[]string](
		"models",
		cache.WithTTL(5*time.Minute),
		cache.WithAsync(true),
		cache.WithClock(now),
	)

	firstGate := make(chan struct{})
	firstDone := make(chan struct{})
	val, found, err := c.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		<-firstGate
		defer close(firstDone)
		return []string{"v1"}, nil
	})
	if err != nil || found || len(val) != 0 || c.HasFailed("k1") {
		t.Fatalf("initial async Get = %v, found=%v, err=%v, failed=%v", val, found, err, c.HasFailed("k1"))
	}

	close(firstGate)
	<-firstDone
	waitForCondition(t, func() bool {
		v, ok, err := c.Get(t.Context(), "k1", nil)
		if err != nil {
			t.Fatal(err)
		}
		return ok && slices.Equal(v, []string{"v1"})
	})

	now.Advance(10 * time.Minute)
	secondGate := make(chan struct{})
	stale, found, err := c.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		<-secondGate
		return []string{"v2"}, nil
	})
	if err != nil || !found || !slices.Equal(stale, []string{"v1"}) {
		t.Fatalf("stale-while-revalidate Get = %v, found=%v, err=%v", stale, found, err)
	}

	close(secondGate)
	waitForCondition(t, func() bool {
		v, ok, err := c.Get(t.Context(), "k1", nil)
		if err != nil {
			t.Fatal(err)
		}
		return ok && slices.Equal(v, []string{"v2"})
	})
}

func TestAsyncGetRecordsFailureAfterBackgroundError(t *testing.T) {
	plugintest.GrantStorage(t)

	var reported atomic.Value
	c := cache.New[[]string]("models", cache.WithAsync(true), cache.WithErrorHandler(func(err error) {
		reported.Store(err.Error())
	}))
	gate := make(chan struct{})

	val, found, err := c.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		<-gate
		return nil, errors.New("load failed")
	})
	if err != nil || found || len(val) != 0 || c.HasFailed("k1") {
		t.Fatalf("in-flight Get = %v, found=%v, err=%v, failed=%v", val, found, err, c.HasFailed("k1"))
	}

	close(gate)
	waitForCondition(t, func() bool { return c.HasFailed("k1") && reported.Load() != nil })
}

func TestDiskPersistenceAcrossTwoCacheInstances(t *testing.T) {
	dir := t.TempDir()
	c1 := cache.New[[]string]("models", cache.WithDir(dir))
	got1, found1, err1 := c1.Get(t.Context(), "agent-a", func(context.Context) ([]string, error) {
		return []string{"alpha", "beta"}, nil
	})
	if err1 != nil || !found1 || !slices.Equal(got1, []string{"alpha", "beta"}) {
		t.Fatalf("c1.Get = %v, %v, %v", got1, found1, err1)
	}

	c2 := cache.New[[]string]("models", cache.WithDir(dir), cache.WithAsync(true))
	refreshGate := make(chan struct{})
	got2, found2, err2 := c2.Get(t.Context(), "agent-a", func(context.Context) ([]string, error) {
		<-refreshGate
		return []string{"gamma"}, nil
	})
	if err2 != nil || !found2 || !slices.Equal(got2, []string{"alpha", "beta"}) {
		t.Fatalf("c2.Get from disk = %v, %v, %v", got2, found2, err2)
	}
	close(refreshGate)
	waitForCondition(t, func() bool {
		v, ok, err := c2.Get(t.Context(), "agent-a", nil)
		if err != nil {
			t.Fatal(err)
		}
		return ok && slices.Equal(v, []string{"gamma"})
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

func TestErrorHandlerReceivesDiskWriteError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	var reported []error
	c := cache.New[[]string](
		"models",
		cache.WithDir(filepath.Join(blocker, "sub")),
		cache.WithDisk(true),
		cache.WithErrorHandler(func(err error) { reported = append(reported, err) }),
	)

	c.Put("k1", []string{"m1"})

	if len(reported) != 1 {
		t.Fatalf("reported %d errors, want 1: %v", len(reported), reported)
	}
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

func TestAMemoryCacheOfAPluginWithoutADirectoryReportsTheDiskFailure(t *testing.T) {
	t.Setenv(cache.EnvCacheDir, "")
	var reported []error
	c := cache.New[[]string]("models", cache.WithDisk(true), cache.WithErrorHandler(func(err error) { reported = append(reported, err) }))

	c.Put("k", []string{"v"})

	if len(reported) != 1 || !strings.Contains(reported[0].Error(), cache.EnvCacheDir) {
		t.Errorf("reported %v, want one error mentioning %s", reported, cache.EnvCacheDir)
	}
}

func TestTheDiskCacheIsStoredInTheGrantedDirectory(t *testing.T) {
	cacheDir, _ := plugintest.GrantStorage(t)
	first := cache.New[[]string]("models", cache.WithDisk(true))
	first.Put("k", []string{"v"})

	second := cache.New[[]string]("models", cache.WithDisk(true))
	got, found, err := second.Get(t.Context(), "k", nil)

	if err != nil || !found || !slices.Equal(got, []string{"v"}) {
		t.Errorf("Get() = %v, %v, %v, want the entry the first cache wrote", got, found, err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "models.json")); err != nil {
		t.Errorf("the cache file is not in the granted directory: %v", err)
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

	dir := t.TempDir()
	unnamed := cache.New[[]string]("", cache.WithDir(dir), cache.WithDisk(true))
	unnamed.Put("k", []string{"v"})
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Fatalf("expected unnamed cache not to write to disk, got %v (%v)", entries, err)
	}

	diskCache := cache.New[[]string]("models", cache.WithDir(dir), cache.WithDisk(true))
	diskCache.Put("empty", []string{})
	if _, err := os.Stat(filepath.Join(dir, "models.json")); err == nil {
		t.Fatal("expected empty slice not to be persisted to disk")
	}

	var corruptErrors []error
	corruptCache := cache.New[[]string]("models", cache.WithDir(dir), cache.WithDisk(true), cache.WithErrorHandler(func(err error) {
		corruptErrors = append(corruptErrors, err)
	}))
	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"k1":{"value":["corrupt"]},"k2":bad}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, ok, err := corruptCache.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		return []string{"fresh"}, nil
	})
	if err != nil || !ok || !slices.Equal(got, []string{"fresh"}) {
		t.Fatalf("expected corrupt disk JSON to be ignored on Get, got %v, %v, %v", got, ok, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"k1":{"value":["corrupt"]},"k2":bad}`), 0o600); err != nil {
		t.Fatal(err)
	}
	corruptCache.Put("k3", []string{"valid"})
	if len(corruptErrors) == 0 {
		t.Fatal("expected error handler to be notified of corrupt disk file on Put")
	}

	now := newFakeClock(time.Unix(5000, 0))
	var nilLoadErrs atomic.Int32
	memAsync := cache.New[string]("", cache.WithAsync(true), cache.WithTTL(time.Second), cache.WithClock(now), cache.WithErrorHandler(func(error) {
		nilLoadErrs.Add(1)
	}))
	memAsync.Put("a", "stale-val")
	now.Advance(5 * time.Second)
	if v, found, err := memAsync.Get(t.Context(), "a", nil); err != nil || !found || v != "stale-val" || memAsync.HasFailed("a") {
		t.Fatalf("expected stale in-memory entry without disk or loader, got %q, found=%v, err=%v, failed=%v", v, found, err, memAsync.HasFailed("a"))
	}
	time.Sleep(10 * time.Millisecond)
	if nilLoadErrs.Load() != 0 || memAsync.HasFailed("a") {
		t.Fatalf("expected nil async loader not to run or fail, got errs=%d failed=%v", nilLoadErrs.Load(), memAsync.HasFailed("a"))
	}

	corruptReadCache := cache.New[[]string]("models", cache.WithDir(dir))
	if v, found, err := corruptReadCache.Get(t.Context(), "k1", func(context.Context) ([]string, error) {
		return []string{"fresh-after-corrupt"}, nil
	}); err != nil || !found || len(v) != 1 || v[0] != "fresh-after-corrupt" {
		t.Fatalf("expected corrupt disk read to fall back to loader, got %v, %v, %v", v, found, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "models.json"), []byte(`{"empty":{"value":[]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyReadCache := cache.New[[]string]("models", cache.WithDir(dir))
	if v, found, err := emptyReadCache.Get(t.Context(), "empty", func(context.Context) ([]string, error) {
		return []string{"fresh-after-empty"}, nil
	}); err != nil || !found || len(v) != 1 || v[0] != "fresh-after-empty" {
		t.Fatalf("expected empty slice on disk to be treated as miss, got %v, %v, %v", v, found, err)
	}

	var asyncCalls atomic.Int32
	gate := make(chan struct{})
	started := make(chan struct{})
	diskAsync := cache.New[[]string]("inflight", cache.WithDir(dir), cache.WithAsync(true), cache.WithTTL(time.Second), cache.WithClock(now))
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
	if err := os.WriteFile(filepath.Join(dir, "inflight.json"), []byte(`{"miss":{"value":["from-disk"]}}`), 0o600); err != nil {
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

	now.Advance(5 * time.Second)
	staleGate := make(chan struct{})
	staleStarted := make(chan struct{})
	var staleCalls atomic.Int32
	if _, _, err := diskAsync.Get(t.Context(), "miss", func(context.Context) ([]string, error) {
		if staleCalls.Add(1) == 1 {
			close(staleStarted)
		}
		<-staleGate
		return []string{"revalidated"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	<-staleStarted
	if _, _, err := diskAsync.Get(t.Context(), "miss", func(context.Context) ([]string, error) {
		staleCalls.Add(1)
		return []string{"duplicate"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	close(staleGate)
	waitForCondition(t, func() bool {
		v, ok, err := diskAsync.Get(t.Context(), "miss", nil)
		return err == nil && ok && slices.Equal(v, []string{"revalidated"})
	})
	if staleCalls.Load() != 1 {
		t.Fatalf("expected 1 stale revalidation call, got %d", staleCalls.Load())
	}

	var zeroFailCache cache.Cache[string]
	if _, _, err := zeroFailCache.Get(t.Context(), "fail", func(context.Context) (string, error) {
		return "", errors.New("fail")
	}); err == nil || !zeroFailCache.HasFailed("fail") {
		t.Fatalf("expected zeroFailCache error and HasFailed=true, got %v", err)
	}

	nilClockTTL := cache.New[string]("nil-clock", cache.WithTTL(time.Hour))
	nilClockTTL.SetClock(nil)
	nilClockTTL.Put("k", "v")
	if v, found, err := nilClockTTL.Get(t.Context(), "k", nil); err != nil || !found || v != "v" {
		t.Fatalf("nilClockTTL with TTL Get = %q, %v, %v", v, found, err)
	}

	var unnamedErrs []error
	unnamedDisk := cache.New[string]("", cache.WithDisk(true), cache.WithDir(dir), cache.WithErrorHandler(func(err error) {
		unnamedErrs = append(unnamedErrs, err)
	}))
	unnamedDisk.Put("k", "v")
	if len(unnamedErrs) != 0 {
		t.Fatalf("expected unnamed cache not to attempt disk write, got errors %v", unnamedErrs)
	}
}

func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}

func TestADiskEntryPastItsTTLIsLoadedAgainAfterARestart(t *testing.T) {
	dir := t.TempDir()
	now := newFakeClock(time.Unix(6000, 0))
	before := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithClock(now))
	before.Put("k", []string{"old"})
	now.Advance(2 * time.Hour)

	after := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithClock(now))
	var calls int
	got, found, err := after.Get(t.Context(), "k", func(context.Context) ([]string, error) {
		calls++
		return []string{"new"}, nil
	})

	if err != nil || !found || !slices.Equal(got, []string{"new"}) || calls != 1 {
		t.Errorf("Get() = %v, %v, %v (calls=%d), want the list loaded again", got, found, err, calls)
	}
}

func TestADiskEntryWithinItsTTLIsKeptAfterARestart(t *testing.T) {
	dir := t.TempDir()
	now := newFakeClock(time.Unix(6000, 0))
	before := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithClock(now))
	before.Put("k", []string{"kept"})
	now.Advance(30 * time.Minute)

	after := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithClock(now))
	var calls int
	got, found, err := after.Get(t.Context(), "k", func(context.Context) ([]string, error) {
		calls++
		return []string{"new"}, nil
	})

	if err != nil || !found || !slices.Equal(got, []string{"kept"}) || calls != 0 {
		t.Errorf("Get() = %v, %v, %v (calls=%d), want the list from disk", got, found, err, calls)
	}
}

func TestAnAsyncDiskEntryPastItsTTLIsRefreshedAfterAReadWithoutALoader(t *testing.T) {
	dir := t.TempDir()
	now := newFakeClock(time.Unix(6000, 0))
	before := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	before.Put("k", []string{"old"})
	now.Advance(2 * time.Hour)

	after := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	if got, found, err := after.Get(t.Context(), "k", nil); err != nil || !found || !slices.Equal(got, []string{"old"}) {
		t.Fatalf("Get() without a loader = %v, %v, %v, want the list from disk", got, found, err)
	}
	if _, _, err := after.Get(t.Context(), "k", func(context.Context) ([]string, error) {
		return []string{"new"}, nil
	}); err != nil {
		t.Fatal(err)
	}

	waitForCondition(t, func() bool {
		got, found, err := after.Get(t.Context(), "k", nil)
		return err == nil && found && slices.Equal(got, []string{"new"})
	})
}

func TestAnAsyncDiskEntryWithinItsTTLIsServedWithoutALoad(t *testing.T) {
	dir := t.TempDir()
	now := newFakeClock(time.Unix(6000, 0))
	before := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	before.Put("k", []string{"kept"})
	now.Advance(30 * time.Minute)

	after := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	var calls atomic.Int32
	got, found, err := after.Get(t.Context(), "k", func(context.Context) ([]string, error) {
		calls.Add(1)
		return []string{"new"}, nil
	})
	time.Sleep(20 * time.Millisecond)

	if err != nil || !found || !slices.Equal(got, []string{"kept"}) || calls.Load() != 0 {
		t.Errorf("Get() = %v, %v, %v (calls=%d), want the list from disk and no load", got, found, err, calls.Load())
	}
}

func TestAnAsyncDiskEntryPastItsTTLIsServedAndRefreshedOnce(t *testing.T) {
	dir := t.TempDir()
	now := newFakeClock(time.Unix(6000, 0))
	before := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	before.Put("k", []string{"old"})
	now.Advance(2 * time.Hour)

	after := cache.New[[]string]("models", cache.WithDir(dir), cache.WithTTL(time.Hour), cache.WithAsync(true), cache.WithClock(now))
	var calls atomic.Int32
	load := func(context.Context) ([]string, error) {
		calls.Add(1)
		return []string{"new"}, nil
	}
	got, found, err := after.Get(t.Context(), "k", load)
	if err != nil || !found || !slices.Equal(got, []string{"old"}) {
		t.Fatalf("Get() = %v, %v, %v, want the list from disk", got, found, err)
	}

	waitForCondition(t, func() bool {
		v, ok, err := after.Get(t.Context(), "k", load)
		return err == nil && ok && slices.Equal(v, []string{"new"})
	})
	if calls.Load() != 1 {
		t.Errorf("loaded %d times, want once", calls.Load())
	}
}
