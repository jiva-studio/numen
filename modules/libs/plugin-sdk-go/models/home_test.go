package models

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetHomeNamesTheEnvironmentFolder(t *testing.T) {
	t.Setenv(EnvDir, "/srv/models")
	got, err := GetHome()
	if err != nil || got != "/srv/models" {
		t.Fatalf("GetHome() = %q, %v", got, err)
	}
}

func TestGetHomeIsUnderTheCacheDirectory(t *testing.T) {
	cache := t.TempDir()
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("HOME", cache)
	got, err := GetHome()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "numen/models") || !strings.HasPrefix(got, cache) {
		t.Fatalf("GetHome() = %q", got)
	}
}

func TestIsOffline(t *testing.T) {
	for value, want := range map[string]bool{
		"": false, "0": false, "no": false, "1": true, "true": true, " TRUE ": true, "yes": true, "on": true,
	} {
		t.Setenv(EnvOffline, value)
		if got := IsOffline(); got != want {
			t.Errorf("%s=%q: IsOffline() = %v, want %v", EnvOffline, value, got, want)
		}
	}
}

func TestGetQuota(t *testing.T) {
	for value, want := range map[string]int64{
		"": DefaultQuota, "x": DefaultQuota, "0": DefaultQuota, "-5": DefaultQuota, "1024": 1024, " 2048 ": 2048,
	} {
		t.Setenv(EnvQuota, value)
		if got := GetQuota(); got != want {
			t.Errorf("%s=%q: GetQuota() = %d, want %d", EnvQuota, value, got, want)
		}
	}
}

func TestGetUsageSumsNestedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, size := range map[string]int{"one": 3, "a/two": 5} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := GetUsage(dir); got != 8 {
		t.Fatalf("GetUsage() = %d, want 8", got)
	}
	if got := GetUsage(filepath.Join(dir, "missing")); got != 0 {
		t.Fatalf("GetUsage(missing) = %d, want 0", got)
	}
}

func TestCheckQuota(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, 6), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvQuota, "10")
	if err := CheckQuota(dir, 4); err != nil {
		t.Fatalf("10 of 10 bytes: %v", err)
	}
	if err := CheckQuota(dir, 5); err == nil {
		t.Fatal("11 of 10 bytes passed")
	}
}
