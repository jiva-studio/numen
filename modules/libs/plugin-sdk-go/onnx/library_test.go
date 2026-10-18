package onnx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindLibraryReturnsTheLibraryInTheFolder(t *testing.T) {
	dir := t.TempDir()
	at := filepath.Join(dir, getLibraryName(runtime.GOOS))
	if err := os.WriteFile(at, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindLibrary(dir); got != at {
		t.Errorf("FindLibrary() = %q, want %q", got, at)
	}
}

func TestFindLibraryOfAFolderWithoutOneIsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, getLibraryName(runtime.GOOS)), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{"", t.TempDir(), dir} {
		if got := FindLibrary(folder); got != "" {
			t.Errorf("FindLibrary(%q) = %q, want empty", folder, got)
		}
	}
}

func TestEachPlatformNamesItsLibrary(t *testing.T) {
	for goos, want := range map[string]string{"linux": "libonnxruntime.so", "darwin": "libonnxruntime.dylib", "windows": "onnxruntime.dll"} {
		if got := getLibraryName(goos); got != want {
			t.Errorf("getLibraryName(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestGetLibraryNameIsThisPlatformsName(t *testing.T) {
	if GetLibraryName() != getLibraryName(runtime.GOOS) {
		t.Errorf("GetLibraryName() = %q", GetLibraryName())
	}
}

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
