package onnx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func setCacheHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("LocalAppData", filepath.Join(home, "local"))
}

func TestGetCoreMLCacheDirIsMadeForThisUserAlone(t *testing.T) {
	setCacheHome(t)
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("stat %s = %v, %v", dir, info, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, want 0700", info.Mode().Perm())
	}
}

func TestDeleteCoreMLCacheEmptiesTheFolderAndReturnsWhetherItContainedAnything(t *testing.T) {
	setCacheHome(t)
	if DeleteCoreMLCache() {
		t.Error("an empty cache held something")
	}
	dir, err := GetCoreMLCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "model.mlmodelc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if !DeleteCoreMLCache() {
		t.Error("a cache with a compiled model held nothing")
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
		t.Errorf("the cache holds %v, %v after it was emptied", entries, err)
	}
}

func TestGetCoreMLOptionsPlacesADefaultModelOnAllUnits(t *testing.T) {
	got := GetCoreMLOptions("/cache", ComputeUnitsDefault)
	want := [][2]string{
		{"ModelCacheDirectory", "/cache"},
		{"EnableOnSubgraphs", "1"},
		{"CreateMLProgram", "1"},
		{"MLComputeUnits", "ALL"},
	}
	if len(got) != len(want) {
		t.Fatalf("GetCoreMLOptions() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("option %d = %v, want %v", i, got[i], want[i])
		}
	}
	if units := GetCoreMLOptions("/cache", ComputeUnitsCPUOnly)[3][1]; units != "CPUOnly" {
		t.Errorf("MLComputeUnits = %q, want CPUOnly", units)
	}
}
