package onnxruntime

import (
	"os"
	"path/filepath"
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
	if info.Mode().Perm() != 0o700 {
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
