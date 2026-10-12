package onnxruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func useCacheHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
}

func TestGetCoreMLCacheDirIsPrivate(t *testing.T) {
	useCacheHome(t)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Skip(err)
	}
	if err := os.MkdirAll(filepath.Join(cache, "numen", "coreml"), 0o755); err != nil {
		t.Fatal(err)
	}

	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(cache, "numen", "coreml"); dir != want {
		t.Errorf("dir = %s, want %s", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 { // Windows reports no permission bits
		t.Errorf("mode = %v, want 0700", info.Mode().Perm())
	}
}

func TestDeleteCoreMLCache(t *testing.T) {
	useCacheHome(t)
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if deleteCoreMLCache() {
		t.Error("an empty cache reported as having held something")
	}
	if err := os.MkdirAll(filepath.Join(dir, "123", "model.mlmodelc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if !deleteCoreMLCache() {
		t.Error("a held cache reported as empty")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("%d entries left", len(entries))
	}
}

func TestGetCoreMLCacheDirFailsWithoutACacheDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the cache directory comes from the profile")
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	dir, err := GetCoreMLCacheDir()
	if err == nil || dir != "" {
		t.Errorf("GetCoreMLCacheDir = %q, %v; want no folder and an error", dir, err)
	}
}

func TestGetCoreMLCacheDirFailsWhereAFolderCannotBeMade(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	blocker := filepath.Join(home, "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", blocker)
	t.Setenv("LocalAppData", blocker)

	dir, err := GetCoreMLCacheDir()
	if err == nil || dir != "" {
		t.Errorf("GetCoreMLCacheDir = %q, %v; want no folder and an error", dir, err)
	}
}

func TestGetCoreMLCacheDirFailsWhereTheFolderIsNotTheUsers(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permissions are not checked here")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	cache := filepath.Join(home, "cache")
	t.Setenv("XDG_CACHE_HOME", cache)
	if err := os.MkdirAll(filepath.Join(cache, "numen"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/", filepath.Join(cache, "numen", "coreml")); err != nil {
		t.Fatal(err)
	}

	dir, err := GetCoreMLCacheDir()
	if err == nil || dir != "" {
		t.Errorf("GetCoreMLCacheDir = %q, %v; want no folder and an error", dir, err)
	}
}

func TestDeleteCoreMLCacheKeepsWhatItCannotRemove(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("permissions are not checked here")
	}
	useCacheHome(t)
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	held := filepath.Join(dir, "123")
	if err := os.MkdirAll(filepath.Join(held, "model.mlmodelc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(held, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(held, 0o700) })

	if deleteCoreMLCache() {
		t.Error("a cache that could not be emptied reported as emptied")
	}
}

func TestDeleteCoreMLCacheReportsNothingHeldWithoutACacheDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the cache directory comes from the profile")
	}
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")

	if deleteCoreMLCache() {
		t.Error("a missing cache directory reported as having held something")
	}
}

func TestGetCoreMLCacheDirCreatesEveryFolderItMakesPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows reports no permission bits")
	}
	useCacheHome(t)

	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{dir, filepath.Dir(dir)} {
		info, err := os.Stat(folder)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Errorf("%s mode = %v, want 0700", folder, info.Mode().Perm())
		}
	}
}
