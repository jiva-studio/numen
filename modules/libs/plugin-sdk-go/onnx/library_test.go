package onnx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindInstalledFindsTheFilesOnTheLoaderPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LD_LIBRARY_PATH", dir)
	lib := filepath.Join(dir, "libnumenfixture.so.1")
	if err := os.WriteFile(lib, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "libnumenfixture.so.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := FindInstalled("libnumenfixture.so", "numenfixture-*")
	if len(got) != 1 || got[0] != lib {
		t.Errorf("FindInstalled() = %v, want [%s]", got, lib)
	}
}
