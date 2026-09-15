package onnxruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// candidates is every ONNX Runtime this machine holds without fetching one:
// beside the application, where a fetched one was unpacked, wherever this
// platform keeps its packages, and by the name the loader searches for on its
// own.
//
// A library in the fetch directory is offered only where it carries the sum
// pinned for this platform.
func candidates(s Settings) []string {
	dir, err := CacheDir(s)
	if err != nil {
		dir = ""
	}
	var out []string
	for _, at := range GetPaths(s.Dir, runtimeNames...) {
		if stands(dir, at) {
			out = append(out, at)
		}
	}
	if dir != "" {
		for _, name := range runtimeNames {
			at := filepath.Join(dir, name)
			if stands(dir, at) {
				out = append(out, at)
			}
		}
	}
	out = append(out, findInstalled(runtimeName(), "*-onnxruntime-*")...)
	return append(out, runtimeName())
}

// stands says whether one file is a library this machine may open. A library in
// the fetch directory carries the sum pinned for this platform, and one from
// anywhere else is the machine's own.
func stands(dir, at string) bool {
	if _, err := os.Stat(at); err != nil {
		return false
	}
	if dir == "" || filepath.Dir(at) != filepath.Clean(dir) {
		return true
	}
	found, ok := releases[runtime.GOOS+"/"+runtime.GOARCH]
	return ok && verify(at, found.library) == nil
}

// load is the first library that loads, the one that did, and why each of the
// others would not. A file that is there is not a library that loads.
func load(candidates []string) (*ort.Engine, string, []string) {
	var refused []string
	for _, at := range candidates {
		engine, err := ort.NewEngine(at)
		if err == nil {
			return engine, at, nil
		}
		refused = append(refused, fmt.Sprintf("%s: %v", at, err))
	}
	return nil, "", refused
}

// GetPaths is where a file may be: in the folder the settings name, and in the
// folder the application was installed into.
func GetPaths(dir string, names ...string) []string {
	var out []string
	for _, name := range names {
		if dir != "" {
			out = append(out, filepath.Join(dir, name))
		}
		if self, err := os.Executable(); err == nil {
			out = append(out, filepath.Join(filepath.Dir(self), name))
			out = append(out, filepath.Join(filepath.Dir(self), "models", name))
		}
	}
	return out
}

// directories are where this machine keeps shared libraries.
func directories() []string {
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

// findInstalled is every file on this machine named for one library, on the
// loader's own path and in the package store beside it. The store is searched
// through the package's name because a machine that has one holds nothing on
// the loader's path at all.
func findInstalled(name, pkg string) []string {
	var out []string
	for _, dir := range directories() {
		out = append(out, findMatching(filepath.Join(dir, name+"*"))...)
	}
	return append(out, findMatching(filepath.Join("/nix/store", pkg, "lib", name+"*"))...)
}

// findMatching is the regular files one pattern names.
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
