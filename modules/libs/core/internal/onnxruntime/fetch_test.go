package onnxruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// A file has two minutes to start arriving, and a second more for each 192 KiB
// its length announces.
func TestPatienceGrowsWithTheLengthAnnounced(t *testing.T) {
	const perSecond = 192 * 1024
	cases := []struct {
		name string
		size int64
		want time.Duration
	}{
		{"a thousand seconds of file", perSecond * 1000, 2*time.Minute + 1000*time.Second},
		{"ten seconds of file", perSecond * 10, 2*time.Minute + 10*time.Second},
		{"less than a second of file", perSecond - 1, 2 * time.Minute},
		{"no length announced", 0, 2*time.Minute + 5461*time.Second},
		{"a negative length", -1, 2*time.Minute + 5461*time.Second},
	}
	for _, c := range cases {
		if got := patience(c.size); got != c.want {
			t.Errorf("%s: patience(%d) = %v, want %v", c.name, c.size, got, c.want)
		}
	}
}

// clearCacheEnvironment leaves the platform no place to keep a cache.
func clearCacheEnvironment(t *testing.T) {
	t.Helper()
	for _, name := range []string{"XDG_CACHE_HOME", "HOME", "LocalAppData"} {
		t.Setenv(name, "")
	}
}

// A folder named in the settings is where files are put, as named.
func TestCacheDirIsTheFolderTheSettingsName(t *testing.T) {
	named := filepath.Join(t.TempDir(), "models")
	clearCacheEnvironment(t)
	got, err := CacheDir(Settings{Dir: named})
	if err != nil || got != named {
		t.Fatalf("CacheDir = %q, %v; want %q", got, err, named)
	}
}

// With no folder named, the platform's cache directory holds numen/models.
func TestCacheDirDefaultsToTheUsersCacheDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", home)
	t.Setenv("LocalAppData", home)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("no cache directory here: %v", err)
	}
	got, err := CacheDir(Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "numen", "models"); got != want {
		t.Fatalf("CacheDir = %q, want %q", got, want)
	}
}

// A machine with no cache directory has nowhere to put a file, and says so.
func TestCacheDirFailsWhenThePlatformHasNone(t *testing.T) {
	clearCacheEnvironment(t)
	if _, err := os.UserCacheDir(); err == nil {
		t.Skip("this platform finds a cache directory without any environment")
	}
	got, err := CacheDir(Settings{})
	if err == nil || got != "" {
		t.Fatalf("CacheDir = %q, %v; want no path and an error", got, err)
	}
}

// A file already in the folder is returned without a download, whether or not
// downloads are allowed.
func TestFetchReturnsAFileAlreadyKept(t *testing.T) {
	dir := t.TempDir()
	address := "https://models.example/m/encoder.onnx"
	at := filepath.Join(dir, GetCacheName(address))
	if err := os.WriteFile(at, []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Fetch(t.Context(), Settings{Dir: dir, Section: "indexing.recognition"}, address)
	if err != nil || got != at {
		t.Fatalf("Fetch = %q, %v; want %q", got, err, at)
	}
}

// A file that is not kept, with downloads off, is an error naming the file and
// the two settings that would provide it.
func TestFetchRefusesToDownloadWhenDownloadsAreOff(t *testing.T) {
	dir := t.TempDir()
	address := "https://models.example/m/encoder.onnx"
	got, err := Fetch(t.Context(), Settings{Dir: dir, Section: "indexing.recognition"}, address)
	if err == nil || got != "" {
		t.Fatalf("Fetch = %q, %v; want no path and an error", got, err)
	}
	for _, want := range []string{address, "indexing.recognition.dir", "indexing.recognition.download"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, GetCacheName(address))); err == nil {
		t.Error("a file was created")
	}
}

// With no place to keep files, Fetch fails before looking for one.
func TestFetchFailsWhenThereIsNowhereToKeepFiles(t *testing.T) {
	clearCacheEnvironment(t)
	if _, err := os.UserCacheDir(); err == nil {
		t.Skip("this platform finds a cache directory without any environment")
	}
	got, err := Fetch(t.Context(), Settings{ShouldDownload: true}, "https://models.example/m.onnx")
	if err == nil || got != "" {
		t.Fatalf("Fetch = %q, %v; want no path and an error", got, err)
	}
}

// A cache name is twelve hex digits of the address's sum, a dash, and the file
// the address ends in.
func TestGetCacheNameIsTheSumOfTheAddressAndItsFile(t *testing.T) {
	address := "https://models.example/m/encoder.onnx"
	sum := sha256.Sum256([]byte(address))
	want := hex.EncodeToString(sum[:6]) + "-encoder.onnx"
	if got := GetCacheName(address); got != want {
		t.Fatalf("GetCacheName = %q, want %q", got, want)
	}
	if !regexp.MustCompile(`^[0-9a-f]{12}-encoder\.onnx$`).MatchString(want) {
		t.Fatalf("%q is not twelve hex digits, a dash and the file", want)
	}
}

// Two addresses ending in one file are two names.
func TestGetCacheNameTellsAddressesWithOneFileApart(t *testing.T) {
	a := GetCacheName("https://one.example/model.onnx")
	b := GetCacheName("https://two.example/model.onnx")
	if a == b {
		t.Fatalf("both are %q", a)
	}
}

// An address that ends in no file is kept as model.onnx.
func TestGetCacheNameFallsBackWhenTheAddressEndsInNoFile(t *testing.T) {
	for _, address := range []string{"", ".", "/"} {
		if got := GetCacheName(address); !strings.HasSuffix(got, "-model.onnx") {
			t.Errorf("GetCacheName(%q) = %q, want a name ending in -model.onnx", address, got)
		}
	}
}
