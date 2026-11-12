//go:build (darwin || freebsd || linux || netbsd) && !android

package onnxruntime

import (
	"path/filepath"

	"github.com/ebitengine/purego"
)

func checkLibrary(pos string) error {
	preloadCppRuntime()
	handle, err := purego.Dlopen(pos, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	symbol, err := purego.Dlsym(handle, "OrtGetApiBase")
	if err != nil {
		return err
	}
	return checkAPI(symbol)
}

func preloadCppRuntime() {
	const name = "libstdc++.so.6"
	if _, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
		return
	}
	dirs := append(getMachineDirs(), filepath.Join("/nix/store", "*-gcc-*-lib", "lib"))
	for _, dir := range dirs {
		matches := findMatchingPaths(filepath.Join(dir, name+"*"))
		sortNewestFirst(matches)
		for _, offset := range matches {
			if _, err := purego.Dlopen(offset, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				return
			}
		}
	}
}
