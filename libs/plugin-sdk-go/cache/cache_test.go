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

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cache"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

func TestSyncGetCachesHitsAndExpiresAfterTTL(t *testing.T) {
	now := clock.NewFake(time.Unix(1000, 0))
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
	now := clock.NewFake(time.Unix(2000, 0))
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

	now := clock.NewFake(time.Unix(3000, 0))

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

	if len(reported) != 1 {
		t.Errorf("reported %v, want one error", reported)
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
