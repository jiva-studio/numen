package packagecache_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin-catalog/packagecache"
)

// The cache is what the catalog declares as its package cache.
var _ catalog.PackageCache = (*packagecache.Cache)(nil)

func checksumOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func newCache(t *testing.T) (*packagecache.Cache, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "numen", "packages")
	cache, err := packagecache.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return cache, dir
}

func readAll(t *testing.T, reader io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func listNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestAPackageIsStoredByChecksumAndReadBack(t *testing.T) {
	cache, dir := newCache(t)
	sum := checksumOf("package bytes")

	if err := cache.Put(t.Context(), sum, strings.NewReader("package bytes")); err != nil {
		t.Fatal(err)
	}
	reader, err := cache.Open(t.Context(), sum)

	if err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, reader); got != "package bytes" {
		t.Errorf("content = %q", got)
	}
	if names := listNames(t, dir); len(names) != 1 || names[0] != sum {
		t.Errorf("the folder holds %v, want only the package named by its checksum", names)
	}
}

func TestContentWithAnotherChecksumIsRejectedAndStoresNoFile(t *testing.T) {
	cache, dir := newCache(t)

	err := cache.Put(t.Context(), checksumOf("expected"), strings.NewReader("tampered"))

	if !errors.Is(err, catalog.ErrChecksumMismatch) {
		t.Errorf("error = %v, want ErrChecksumMismatch", err)
	}
	if names := listNames(t, dir); len(names) != 0 {
		t.Errorf("the folder holds %v after a refused package", names)
	}
}

func TestAPackageChangedOnDiskIsCaughtOnUseAndRemoved(t *testing.T) {
	cache, dir := newCache(t)
	sum := checksumOf("package bytes")
	if err := cache.Put(t.Context(), sum, strings.NewReader("package bytes")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, sum), []byte("replaced by another application"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := cache.Open(t.Context(), sum)
	_, again := cache.Open(t.Context(), sum)

	if !errors.Is(err, catalog.ErrChecksumMismatch) {
		t.Errorf("error = %v, want ErrChecksumMismatch", err)
	}
	if !errors.Is(again, catalog.ErrPackageNotCached) {
		t.Errorf("a package that failed the check is still there: error = %v", again)
	}
}

func TestAPackageThatIsNotThereIsNotCached(t *testing.T) {
	cache, _ := newCache(t)

	_, err := cache.Open(t.Context(), checksumOf("absent"))

	if !errors.Is(err, catalog.ErrPackageNotCached) {
		t.Errorf("error = %v, want ErrPackageNotCached", err)
	}
}

func TestAChecksumThatIsNotAHashNamesNoFile(t *testing.T) {
	cache, dir := newCache(t)
	for _, sum := range []string{"", "../escape", strings.ToUpper(checksumOf("x")), checksumOf("x")[:63]} {
		if err := cache.Put(t.Context(), sum, strings.NewReader("x")); !errors.Is(err, catalog.ErrInvalidChecksum) {
			t.Errorf("Put(%q) error = %v, want ErrInvalidChecksum", sum, err)
		}
		if _, err := cache.Open(t.Context(), sum); !errors.Is(err, catalog.ErrInvalidChecksum) {
			t.Errorf("Open(%q) error = %v, want ErrInvalidChecksum", sum, err)
		}
		if err := cache.Remove(t.Context(), sum); !errors.Is(err, catalog.ErrInvalidChecksum) {
			t.Errorf("Remove(%q) error = %v, want ErrInvalidChecksum", sum, err)
		}
	}
	if _, err := os.Stat(dir); err == nil {
		t.Errorf("a refused checksum created the folder")
	}
}

func TestRemovingAPackageThatIsAbsentIsNotAnError(t *testing.T) {
	cache, _ := newCache(t)
	sum := checksumOf("package bytes")
	if err := cache.Put(t.Context(), sum, strings.NewReader("package bytes")); err != nil {
		t.Fatal(err)
	}

	first := cache.Remove(t.Context(), sum)
	second := cache.Remove(t.Context(), sum)

	if first != nil || second != nil {
		t.Errorf("errors = %v, %v", first, second)
	}
	if _, err := cache.Open(t.Context(), sum); !errors.Is(err, catalog.ErrPackageNotCached) {
		t.Errorf("the package is still cached: %v", err)
	}
}

func TestApplicationsStoringTheSamePackageAtOnceLeaveOneWholePackage(t *testing.T) {
	cache, dir := newCache(t)
	content := strings.Repeat("0123456789", 100000)
	sum := checksumOf(content)

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- cache.Put(t.Context(), sum, strings.NewReader(content))
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Put: %v", err)
		}
	}
	reader, err := cache.Open(t.Context(), sum)
	if err != nil {
		t.Fatal(err)
	}
	if got := readAll(t, reader); got != content {
		t.Errorf("the stored package differs in length: %d, want %d", len(got), len(content))
	}
	if names := listNames(t, dir); len(names) != 1 {
		t.Errorf("the folder holds %v after the writes", names)
	}
}

func TestAnEndedContextStopsTheWrite(t *testing.T) {
	cache, dir := newCache(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := cache.Put(ctx, checksumOf("x"), bytes.NewReader([]byte("x")))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(dir); statErr == nil {
		t.Error("the folder was created for a write that did not start")
	}
}

func TestACacheNeedsAFolder(t *testing.T) {
	if _, err := packagecache.New(""); err == nil {
		t.Error("a cache with no folder was made")
	}
}

func TestTheDefaultFolderIsSharedInTheUserCacheFolder(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LocalAppData", t.TempDir())

	dir, err := packagecache.DefaultDir()
	if err != nil {
		t.Skipf("this machine has no user cache folder: %v", err)
	}

	if filepath.Base(dir) != "packages" || filepath.Base(filepath.Dir(dir)) != "numen" {
		t.Errorf("default folder = %s, want .../numen/packages", dir)
	}
}
