package onnx

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestFindLibraryReturnsTheLibraryInTheFolder(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, getLibraryName(runtime.GOOS))
	if err := os.WriteFile(targetPath, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := FindLibrary(dir); got != targetPath {
		t.Errorf("FindLibrary() = %q, want %q", got, targetPath)
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

	cwdLibDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwdLibDir, getLibraryName(runtime.GOOS)), []byte("lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwdLibDir)
	if got := FindLibrary(""); got != "" {
		t.Errorf("FindLibrary(\"\") in dir with library = %q, want empty", got)
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
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("LD_LIBRARY_PATH", dir)
	lib := filepath.Join(dir, "libnumenfixture.so.1")
	if err := os.WriteFile(lib, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "libnumenfixture.so.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	nixProfileLib := filepath.Join(home, ".nix-profile", "lib")
	if err := os.MkdirAll(nixProfileLib, 0o755); err != nil {
		t.Fatal(err)
	}
	nixLib := filepath.Join(nixProfileLib, "libnumenfixture.so.2")
	if err := os.WriteFile(nixLib, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	got := FindInstalled("libnumenfixture.so", "numenfixture-*")
	if len(got) != 2 || got[0] != lib || got[1] != nixLib {
		t.Errorf("FindInstalled() = %v, want [%s %s]", got, lib, nixLib)
	}
}

func TestSortNewestFirstInsertsTheHighestVersionFirst(t *testing.T) {
	paths := []string{
		"/nix/store/aaaa-gcc-15.2.0-lib/lib/libstdc++.so.6",
		"/nix/store/dddd-gcc-15.10.0-lib/lib/libstdc++.so.6",
		"/nix/store/bbbb-gcc-16.2.0-lib/lib/libstdc++.so.6",
		"/nix/store/cccc-gcc-9.5.0-lib/lib/libstdc++.so.6",
	}
	sortNewestFirst(paths)
	want := []string{
		"/nix/store/bbbb-gcc-16.2.0-lib/lib/libstdc++.so.6",
		"/nix/store/dddd-gcc-15.10.0-lib/lib/libstdc++.so.6",
		"/nix/store/aaaa-gcc-15.2.0-lib/lib/libstdc++.so.6",
		"/nix/store/cccc-gcc-9.5.0-lib/lib/libstdc++.so.6",
	}
	if !slices.Equal(paths, want) {
		t.Errorf("sortNewestFirst() = %v, want %v", paths, want)
	}
	if got := getVersion("relative-1.2"); got != nil {
		t.Errorf("getVersion(relative-1.2) = %v, want nil", got)
	}
}
