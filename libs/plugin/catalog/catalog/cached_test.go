package catalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalogtest"
)

func getEpoch() time.Time { return time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC) }

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func TestAnEmptyCacheFillsFromTheSource(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if err != nil || len(entries) != 1 {
		t.Fatalf("List = %v, %v; want the entry of the source", entries, err)
	}
	snapshot, readErr := cache.Read(t.Context())
	if readErr != nil || !snapshot.At.Equal(getEpoch()) || len(snapshot.Entries) != 1 {
		t.Errorf("snapshot = %+v, %v; want the listing taken at %v", snapshot, readErr, getEpoch())
	}
}

func TestAnEmptyCacheAndAFailingSourceReturnTheFailure(t *testing.T) {
	source := catalogtest.NewFakeSource("remote")
	source.Err = errors.New("offline")
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if !errors.Is(err, source.Err) || len(entries) != 0 {
		t.Errorf("List = %v, %v; want no entries and the failure of the source", entries, err)
	}
}

func TestAFreshSnapshotIsNotReadFromTheSourceAgain(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	source.Err = errors.New("offline")
	now.now = getEpoch().Add(59 * time.Minute)

	entries, err := cached.List(t.Context())

	if err != nil || len(entries) != 1 || cache.Writes != 1 {
		t.Errorf("List = %v, %v after %d writes; want the snapshot with no read of the source", entries, err, cache.Writes)
	}
}

func TestAStaleSnapshotAnswersWhenTheSourceCannot(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	source.Err = errors.New("offline")
	now.now = getEpoch().Add(2 * time.Hour)

	entries, err := cached.List(t.Context())

	if len(entries) != 1 || !errors.Is(err, source.Err) {
		t.Errorf("List = %v, %v; want the last snapshot with the failure of the source", entries, err)
	}
}

func TestAnExpiredSnapshotIsReplacedByTheSource(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	now.now = getEpoch().Add(2 * time.Hour)

	_, err := cached.List(t.Context())

	snapshot, readErr := cache.Read(t.Context())
	if err != nil || readErr != nil || cache.Writes != 2 || !snapshot.At.Equal(now.now) {
		t.Errorf("error = %v, writes = %d, snapshot taken at %v; want a second snapshot at %v", err, cache.Writes, snapshot.At, now.now)
	}
	// Exact TTL boundary: age == ttl is expired and triggers a third write
	now.now = now.now.Add(time.Hour)
	if _, err := cached.List(t.Context()); err != nil || cache.Writes != 3 {
		t.Errorf("at exact TTL boundary: err = %v, writes = %d, want 3", err, cache.Writes)
	}
}

func TestAWriteFailureOfTheCacheIsReportedWithTheEntries(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{WriteErr: errors.New("disk full")}
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if len(entries) != 1 || !errors.Is(err, cache.WriteErr) {
		t.Errorf("List = %v, %v; want the entries with the write failure", entries, err)
	}
}

func TestACachedSourceGetsOneEntry(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	now := &clock{now: getEpoch()}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)

	got, err := cached.Get(t.Context(), "org.example.a")
	_, missing := cached.Get(t.Context(), "org.example.b")

	if err != nil || got.ID != "org.example.a" || cached.Name() != "remote" {
		t.Errorf("Get = %+v, %v", got, err)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("Get of a missing plugin: error = %v", missing)
	}

	readFailErr := errors.New("corrupt cache")
	badCache := &errCache{readErr: readFailErr}
	cachedWithBadRead := catalog.NewCachedSource(source, badCache, time.Hour, now.Now)
	got2, err2 := cachedWithBadRead.Get(t.Context(), "org.example.a")
	if got2.ID != "org.example.a" || !errors.Is(err2, readFailErr) {
		t.Errorf("Get with cache read error = %+v, %v; want entry and wrapped readFailErr", got2, err2)
	}
	partErr := errors.New("partial source error")
	partSrc := &partialSource{entries: []catalog.Entry{entry("org.example.a", "1.0.0", catalog.StateAvailable)}, err: partErr}
	badCache.readErr = catalog.ErrEmptyCache
	cachedPartial := catalog.NewCachedSource(partSrc, badCache, time.Hour, now.Now)
	entries3, err3 := cachedPartial.List(t.Context())
	if len(entries3) != 1 || !errors.Is(err3, partErr) {
		t.Errorf("List with partial source error and empty cache = %v, %v; want 1 entry and partErr", entries3, err3)
	}
}

type errCache struct {
	catalogtest.FakeCache
	readErr error
}

func (c *errCache) Read(ctx context.Context) (catalog.Snapshot, error) {
	if c.readErr != nil {
		return catalog.Snapshot{}, c.readErr
	}
	return c.FakeCache.Read(ctx)
}

type partialSource struct {
	entries []catalog.Entry
	err     error
}

func (s *partialSource) Name() string { return "remote" }

func (s *partialSource) List(context.Context) ([]catalog.Entry, error) {
	return s.entries, s.err
}

func (s *partialSource) Get(context.Context, string) (catalog.Entry, error) {
	return s.entries[0], s.err
}

func TestACachedSourceSatisfiesTheSourceContract(t *testing.T) {
	catalogtest.RunSourceContract(t, func(_ *testing.T, entries []catalog.Entry) catalog.Source {
		now := &clock{now: getEpoch()}
		return catalog.NewCachedSource(catalogtest.NewFakeSource("remote", entries...), &catalogtest.FakeCache{}, time.Hour, now.Now)
	}, catalogtest.AvailableEntries())
}
