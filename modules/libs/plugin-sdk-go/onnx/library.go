package onnx

import (
	"os"
	"path/filepath"
	"runtime"
)

// FindInstalled returns every file on this machine named for one library, on
// the loader's own path and in the package store beside it. The store is
// searched through the package's name because a machine that has one holds
// nothing on the loader's path at all.
func FindInstalled(name, pkg string) []string {
	var out []string
	for _, dir := range getLibraryDirectories() {
		out = append(out, findMatching(filepath.Join(dir, name+"*"))...)
	}
	return append(out, findMatching(filepath.Join("/nix/store", pkg, "lib", name+"*"))...)
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
	for _, at := range found {
		if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
			out = append(out, at)
		}
	}
	return out
}
