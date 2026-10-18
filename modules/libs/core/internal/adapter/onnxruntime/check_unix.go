//go:build (darwin || freebsd || linux || netbsd) && !android

package onnxruntime

import (
	"path/filepath"
	"sync"

	"github.com/ebitengine/purego"
)

// checkLibrary opens the library at a path with the platform loader and returns
// an error when it does not open or serves no apiVersion table. The library
// stays loaded.
func checkLibrary(at string) error {
	preloadCppRuntime()
	handle, err := purego.Dlopen(at, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return err
	}
	symbol, err := purego.Dlsym(handle, "OrtGetApiBase")
	if err != nil {
		return err
	}
	return checkAPI(symbol)
}

// preloadCppRuntime loads the C++ runtime into the global namespace when the
// loader does not find it by name, from the folders shared libraries live in or
// the package store. A build published for Linux in general names it and
// expects the system to hold it.
var preloadCppRuntime = sync.OnceFunc(func() {
	const name = "libstdc++.so.6"
	if _, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
		return
	}
	dirs := append(getMachineDirs(), filepath.Join("/nix/store", "*-gcc-*-lib", "lib"))
	for _, dir := range dirs {
		for _, at := range findMatching(filepath.Join(dir, name+"*")) {
			if _, err := purego.Dlopen(at, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
				return
			}
		}
	}
})
