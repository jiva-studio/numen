package onnx

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// or empty when dir is empty or holds none.
func FindLibrary(dir string) string {
	if dir == "" {
		return ""
	}
	targetPath := filepath.Join(dir, GetLibraryName())
	if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
		return targetPath
	}
	return ""
}

// GetLibraryName returns what the library is called on this operating system.
func GetLibraryName() string {
	return getLibraryName(runtime.GOOS)
}

// getLibraryName returns what the library is called on an operating system.
func getLibraryName(goos string) string {
	switch goos {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}

// FindInstalled returns every file on this machine named for one library, on
// the loader's own path and in the package store beside it. The store is
// searched through the package's name because a machine that has one holds
// nothing on the loader's path at all.
func FindInstalled(name, pkg string) []string {
	var out []string
	for _, dir := range getLibraryDirectories() {
		out = append(out, findMatching(filepath.Join(dir, name+"*"))...)
	}
	stored := findMatching(filepath.Join("/nix/store", pkg, "lib", name+"*"))
	sortNewestFirst(stored)
	return append(out, stored...)
}

var versionPart = regexp.MustCompile(`[0-9]+`)

// sortNewestFirst orders package store paths by the numbers in the package name
// that follows the store hash, highest first. The newest package links the
// newest C library, and the library loaded first fixes the C library every
// later one resolves against.
func sortNewestFirst(paths []string) {
	slices.SortStableFunc(paths, func(a, b string) int {
		return -slices.Compare(getVersion(a), getVersion(b))
	})
}

// getVersion returns the numbers in the store package name of a path.
func getVersion(path string) []int {
	rel, err := filepath.Rel("/nix/store", path)
	if err != nil {
		return nil
	}
	pkg, _, _ := strings.Cut(rel, string(filepath.Separator))
	_, name, _ := strings.Cut(pkg, "-")
	var out []int
	for _, digits := range versionPart.FindAllString(name, -1) {
		n, _ := strconv.Atoi(digits)
		out = append(out, n)
	}
	return out
}

// getLibraryDirectories returns where this machine keeps shared libraries.
func getLibraryDirectories() []string {
	out := filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, ".nix-profile", "lib"))
	}
	return append(out,
		"/run/current-system/sw/lib",
		"/usr/local/lib",
		"/usr/lib64",
		"/usr/lib/"+runtime.GOARCH+"-linux-gnu",
		"/usr/lib/x86_64-linux-gnu",
		"/usr/lib/aarch64-linux-gnu",
		"/usr/lib",
		"/lib64",
		"/lib",
	)
}

// findMatching returns the regular files one pattern names.
func findMatching(pattern string) []string {
	found, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range found {
		if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
			out = append(out, at)
		}
	}
	return out
}
