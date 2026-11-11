package onnx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/cache"
)

func grantModel(t *testing.T) (shared, model string) {
	t.Helper()
	shared = t.TempDir()
	t.Setenv(cache.EnvSharedDir, shared)
	t.Setenv(cache.EnvCacheDir, t.TempDir())
	model = filepath.Join(t.TempDir(), "model.onnx")
	if err := os.WriteFile(model, []byte("weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	return shared, model
}

func TestGetCoreMLCacheDirIsMadeForThisUserAlone(t *testing.T) {
	shared, model := grantModel(t)
	dir, err := GetCoreMLCacheDir(model)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(dir)) != shared || filepath.Base(filepath.Dir(dir)) != "coreml" {
		t.Errorf("folder %s is not coreml/<key> in the shared directory %s", dir, shared)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("stat %s = %v, %v", dir, info, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, want 0700", info.Mode().Perm())
	}
}

func TestACompiledModelIsStoredOncePerModelAndNotPerPlugin(t *testing.T) {
	_, model := grantModel(t)
	first, err := GetCoreMLCacheDir(model)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(cache.EnvCacheDir, t.TempDir())

	second, err := GetCoreMLCacheDir(model)

	if err != nil || second != first {
		t.Errorf("GetCoreMLCacheDir() = %q, %v, want %q", second, err, first)
	}
	if err := os.WriteFile(model, []byte("changed weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := GetCoreMLCacheDir(model); err != nil || changed == first {
		t.Errorf("a changed model got the folder %q again, %v", changed, err)
	}
}

func TestAHostThatSharesNothingKeepsTheCompiledModelInThePluginFolder(t *testing.T) {
	_, model := grantModel(t)
	t.Setenv(cache.EnvSharedDir, "")
	own := os.Getenv(cache.EnvCacheDir)

	dir, err := GetCoreMLCacheDir(model)

	if err != nil || filepath.Dir(filepath.Dir(dir)) != own {
		t.Errorf("GetCoreMLCacheDir() = %q, %v, want it inside %s", dir, err, own)
	}
}

func TestDeleteCoreMLCache_EmptiesDirAndReportsContent(t *testing.T) {
	_, model := grantModel(t)
	if DeleteCoreMLCache(model) {
		t.Error("an empty cache held something")
	}
	dir, err := GetCoreMLCacheDir(model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "model.mlmodelc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, lockName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !DeleteCoreMLCache(model) {
		t.Error("a cache with a compiled model held nothing")
	}
	if entries, err := os.ReadDir(dir); err != nil || len(entries) != 1 || entries[0].Name() != lockName {
		t.Errorf("the cache holds %v, %v after it was emptied, want only the lock", entries, err)
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

func TestCoreMLCacheErrors(t *testing.T) {
	_, model := grantModel(t)
	missingModel := filepath.Join(t.TempDir(), "missing.onnx")
	if _, err := GetCoreMLCacheDir(missingModel); err == nil {
		t.Fatal("GetCoreMLCacheDir with missing model succeeded, want error")
	}
	if DeleteCoreMLCache(missingModel) {
		t.Fatal("DeleteCoreMLCache with missing model returned true, want false")
	}

	t.Setenv(cache.EnvSharedDir, model)
	if _, err := GetCoreMLCacheDir(model); err == nil {
		t.Fatal("GetCoreMLCacheDir with file as shared dir succeeded, want error")
	}

	t.Setenv(cache.EnvSharedDir, "")
	t.Setenv(cache.EnvCacheDir, "")
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	if _, err := GetCoreMLCacheDir(model); err == nil {
		t.Fatal("GetCoreMLCacheDir without cache env or HOME succeeded, want error")
	}
}

func TestDeleteCoreMLCacheReturnsFalseWhenRemoveFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("chmod permission check requires non-root unix")
	}
	_, model := grantModel(t)
	dir, err := GetCoreMLCacheDir(model)
	if err != nil {
		t.Fatal(err)
	}
	lockedSub := filepath.Join(dir, "locked-sub")
	if err := os.MkdirAll(lockedSub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lockedSub, "inner"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedSub, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(lockedSub, 0o700); err != nil {
			t.Error(err)
		}
	})
	if DeleteCoreMLCache(model) {
		t.Fatal("DeleteCoreMLCache returned true when removing an entry failed")
	}
}
