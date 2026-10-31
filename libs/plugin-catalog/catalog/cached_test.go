package catalog_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalogtest"
)

var epoch = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func TestAnEmptyCacheFillsFromTheSource(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if err != nil || len(entries) != 1 {
		t.Fatalf("List = %v, %v; want the entry of the source", entries, err)
	}
	snapshot, readErr := cache.Read(t.Context())
	if readErr != nil || !snapshot.At.Equal(epoch) || len(snapshot.Entries) != 1 {
		t.Errorf("snapshot = %+v, %v; want the listing taken at %v", snapshot, readErr, epoch)
	}
}

func TestAnEmptyCacheAndAFailingSourceReturnTheFailure(t *testing.T) {
	source := catalogtest.NewFakeSource("remote")
	source.Err = errors.New("offline")
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if !errors.Is(err, source.Err) || len(entries) != 0 {
		t.Errorf("List = %v, %v; want no entries and the failure of the source", entries, err)
	}
}

func TestAFreshSnapshotIsNotReadFromTheSourceAgain(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	source.Err = errors.New("offline")
	now.now = epoch.Add(59 * time.Minute)

	entries, err := cached.List(t.Context())

	if err != nil || len(entries) != 1 || cache.Writes != 1 {
		t.Errorf("List = %v, %v after %d writes; want the snapshot with no read of the source", entries, err, cache.Writes)
	}
}

func TestAStaleSnapshotAnswersWhenTheSourceCannot(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	source.Err = errors.New("offline")
	now.now = epoch.Add(2 * time.Hour)

	entries, err := cached.List(t.Context())

	if len(entries) != 1 || !errors.Is(err, source.Err) {
		t.Errorf("List = %v, %v; want the last snapshot with the failure of the source", entries, err)
	}
}

func TestAnExpiredSnapshotIsReplacedByTheSource(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{}
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)
	if _, err := cached.List(t.Context()); err != nil {
		t.Fatal(err)
	}
	now.now = epoch.Add(2 * time.Hour)

	_, err := cached.List(t.Context())

	snapshot, readErr := cache.Read(t.Context())
	if err != nil || readErr != nil || cache.Writes != 2 || !snapshot.At.Equal(now.now) {
		t.Errorf("error = %v, writes = %d, snapshot taken at %v; want a second snapshot at %v", err, cache.Writes, snapshot.At, now.now)
	}
}

func TestAWriteFailureOfTheCacheIsReportedWithTheEntries(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	cache := &catalogtest.FakeCache{WriteErr: errors.New("disk full")}
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, cache, time.Hour, now.Now)

	entries, err := cached.List(t.Context())

	if len(entries) != 1 || !errors.Is(err, cache.WriteErr) {
		t.Errorf("List = %v, %v; want the entries with the write failure", entries, err)
	}
}

func TestACachedSourceGetsOneEntry(t *testing.T) {
	source := catalogtest.NewFakeSource("remote", entry("org.example.a", "1.0.0", catalog.StateAvailable))
	now := &clock{now: epoch}
	cached := catalog.NewCachedSource(source, &catalogtest.FakeCache{}, time.Hour, now.Now)

	got, err := cached.Get(t.Context(), "org.example.a")
	_, missing := cached.Get(t.Context(), "org.example.b")

	if err != nil || got.ID != "org.example.a" || cached.Name() != "remote" {
		t.Errorf("Get = %+v, %v", got, err)
	}
	if !errors.Is(missing, catalog.ErrNotFound) {
		t.Errorf("Get of a missing plugin: error = %v", missing)
	}
}

func TestACachedSourceSatisfiesTheSourceContract(t *testing.T) {
	catalogtest.RunSourceContract(t, func(_ *testing.T, entries []catalog.Entry) catalog.Source {
		now := &clock{now: epoch}
		return catalog.NewCachedSource(catalogtest.NewFakeSource("remote", entries...), &catalogtest.FakeCache{}, time.Hour, now.Now)
	}, catalogtest.OfferedEntries())
}
