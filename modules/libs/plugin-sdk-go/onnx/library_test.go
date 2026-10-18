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
