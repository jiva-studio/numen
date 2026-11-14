package onnx

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// grantModel returns a cache root the caller hands to the module and a model file outside it.
func grantModel(t *testing.T) (root, model string) {
	t.Helper()
	root = t.TempDir()
	model = filepath.Join(t.TempDir(), "model.onnx")
	if err := os.WriteFile(model, []byte("weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, model
}

func TestGetCoreMLCacheDirIsMadeForThisUserAlone(t *testing.T) {
	root, model := grantModel(t)
	dir, err := GetCoreMLCacheDir(root, model)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(dir)) != root || filepath.Base(filepath.Dir(dir)) != "coreml" {
		t.Errorf("folder %s is not coreml/<key> in the cache root %s", dir, root)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("stat %s = %v, %v", dir, info, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		t.Errorf("mode = %v, want 0700", info.Mode().Perm())
	}
}

// The plugin resolves the cache root and passes it; the module reads no variable of the host to find one.
func TestGetCoreMLCacheDirKeepsTheCompiledModelInTheGivenRoot(t *testing.T) {
	root, model := grantModel(t)
	t.Setenv("HOME", "")
	t.Setenv("XDG_CACHE_HOME", "")
	t.Setenv("NUMEN_PLUGIN_CACHE_DIR", t.TempDir())
	t.Setenv("NUMEN_SHARED_CACHE_DIR", t.TempDir())

	dir, err := GetCoreMLCacheDir(root, model)

	if err != nil || !strings.HasPrefix(dir, root+string(filepath.Separator)) {
		t.Errorf("GetCoreMLCacheDir(%q) = %q, %v, want a folder inside the root", root, dir, err)
	}
	other := t.TempDir()
	if moved, err := GetCoreMLCacheDir(other, model); err != nil || !strings.HasPrefix(moved, other+string(filepath.Separator)) {
		t.Errorf("GetCoreMLCacheDir(%q) = %q, %v, want a folder inside that root", other, moved, err)
	}
}

func TestACompiledModelIsStoredOncePerModel(t *testing.T) {
	root, model := grantModel(t)
	first, err := GetCoreMLCacheDir(root, model)
	if err != nil {
		t.Fatal(err)
	}

	second, err := GetCoreMLCacheDir(root, model)

	if err != nil || second != first {
		t.Errorf("GetCoreMLCacheDir() = %q, %v, want %q", second, err, first)
	}
	if err := os.WriteFile(model, []byte("changed weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	if changed, err := GetCoreMLCacheDir(root, model); err != nil || changed == first {
		t.Errorf("a changed model got the folder %q again, %v", changed, err)
	}
}

// A provisioned model is keyed by the sha256 its .ready file records, the key the plugin SDK gives
// the same file, so the folder stays the same when the module takes over the key from the SDK.
func TestAProvisionedModelIsKeyedByItsPinnedSum(t *testing.T) {
	root, model := grantModel(t)
	sum := strings.Repeat("ab", 32)
	ready := "model.onnx\n" + strings.ToUpper(sum) + "\n"
	if err := os.WriteFile(filepath.Join(filepath.Dir(model), ".ready"), []byte(ready), 0o600); err != nil {
		t.Fatal(err)
	}

	dir, err := GetCoreMLCacheDir(root, model)

	if want := filepath.Join(root, "coreml", sum); err != nil || dir != want {
		t.Errorf("GetCoreMLCacheDir() = %q, %v, want %q", dir, err, want)
	}
}

func TestDeleteCoreMLCache_EmptiesDirAndReportsContent(t *testing.T) {
	root, model := grantModel(t)
	if DeleteCoreMLCache(root, model) {
		t.Error("an empty cache held something")
	}
	dir, err := GetCoreMLCacheDir(root, model)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "model.mlmodelc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, lockName), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if !DeleteCoreMLCache(root, model) {
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
	root, model := grantModel(t)
	missingModel := filepath.Join(t.TempDir(), "missing.onnx")
	if _, err := GetCoreMLCacheDir(root, missingModel); err == nil {
		t.Fatal("GetCoreMLCacheDir with missing model succeeded, want error")
	}
	if DeleteCoreMLCache(root, missingModel) {
		t.Fatal("DeleteCoreMLCache with missing model returned true, want false")
	}
	if _, err := GetCoreMLCacheDir(model, model); err == nil {
		t.Fatal("GetCoreMLCacheDir with a file as the cache root succeeded, want error")
	}
	if _, err := GetCoreMLCacheDir("", model); err == nil {
		t.Fatal("GetCoreMLCacheDir with an empty cache root succeeded, want error")
	}
}

func TestDeleteCoreMLCacheReturnsFalseWhenRemoveFails(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("chmod permission check requires non-root unix")
	}
	root, model := grantModel(t)
	dir, err := GetCoreMLCacheDir(root, model)
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
	if DeleteCoreMLCache(root, model) {
		t.Fatal("DeleteCoreMLCache returned true when removing an entry failed")
	}
}
