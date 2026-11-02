package snapshotcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/catalogtest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/snapshotcache"
)

var _ catalog.Cache = (*snapshotcache.Cache)(nil)

func newCache(t *testing.T) *snapshotcache.Cache {
	t.Helper()
	cache, err := snapshotcache.New(filepath.Join(t.TempDir(), "numen", "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	return cache
}

func TestAFileThatDoesNotExistIsAnEmptyCache(t *testing.T) {
	_, err := newCache(t).Read(t.Context())

	if !errors.Is(err, catalog.ErrEmptyCache) {
		t.Errorf("error = %v, want ErrEmptyCache", err)
	}
}

func TestASnapshotIsReadBackWithItsTimeAndEntries(t *testing.T) {
	cache := newCache(t)
	takenAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	entries := catalogtest.OfferedEntries()
	for i := range entries {
		entries[i].Source = "remote"
	}

	if err := cache.Write(t.Context(), catalog.Snapshot{Entries: entries, At: takenAt}); err != nil {
		t.Fatal(err)
	}
	got, err := cache.Read(t.Context())

	if err != nil || !got.At.Equal(takenAt) || len(got.Entries) != len(entries) {
		t.Fatalf("snapshot = %+v, %v", got, err)
	}
	for i, entry := range got.Entries {
		want := entries[i]
		if entry.ID != want.ID || entry.Version != want.Version || entry.State != want.State || entry.Source != "remote" ||
			entry.Contributes.Feature != want.Contributes.Feature || len(entry.Contributes.TabKinds) != 1 ||
			entry.Contributes.TabKinds[0] != want.Contributes.TabKinds[0] || entry.Requirements.Targets[0] != want.Requirements.Targets[0] {
			t.Errorf("entry %d = %+v, want %+v", i, entry, want)
		}
	}
}

func TestASnapshotReplacesThePreviousAndRemovesItsTemporaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	cache, err := snapshotcache.New(path)
	if err != nil {
		t.Fatal(err)
	}
	takenAt := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	for i := range 2 {
		if err := cache.Write(t.Context(), catalog.Snapshot{Entries: catalogtest.OfferedEntries()[:i+1], At: takenAt}); err != nil {
			t.Fatal(err)
		}
	}
	got, readErr := cache.Read(t.Context())

	if readErr != nil || len(got.Entries) != 2 {
		t.Errorf("snapshot = %+v, %v; want the second write", got, readErr)
	}
	files, dirErr := os.ReadDir(filepath.Dir(path))
	if dirErr != nil || len(files) != 1 {
		t.Errorf("the folder holds %v, %v; want the snapshot alone", files, dirErr)
	}
}

func TestAFileThatIsNotASnapshotIsAnErrorAndNotAnEmptyCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	cache, err := snapshotcache.New(path)
	if err != nil {
		t.Fatal(err)
	}

	_, readErr := cache.Read(t.Context())

	if readErr == nil || errors.Is(readErr, catalog.ErrEmptyCache) {
		t.Errorf("error = %v, want a decode error", readErr)
	}
}

func TestACacheNeedsAFile(t *testing.T) {
	if _, err := snapshotcache.New(""); err == nil {
		t.Error("a cache with no file was made")
	}
}
