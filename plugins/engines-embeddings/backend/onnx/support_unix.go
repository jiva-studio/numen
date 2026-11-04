//go:build darwin || freebsd || linux || netbsd

package onnx

import "github.com/ebitengine/purego"

// preloadCppRuntime loads what the runtime is linked against and the loader cannot find.
//
// A build published for Linux in general names the C++ runtime and expects the
// system to hold it. Loading it here first, into the global namespace, is what
// resolves that name on a machine that keeps the file somewhere else.
func preloadCppRuntime() {
	// gomutants:disable-next-line RANGE_BREAK reason="Preloading shared libraries into the global linker namespace is a platform side effect."
	for _, p := range FindInstalled("libstdc++.so", "*-gcc-*-lib") {
		// gomutants:disable-next-line INVERT_BITWISE,CONDITIONALS_NEGATION,BRANCH_IF reason="Preloading shared libraries into the global linker namespace is a platform side effect."
		if _, err := purego.Dlopen(p, purego.RTLD_NOW|purego.RTLD_GLOBAL); err == nil {
			return
		}
	}
}

func loadLibrary(name string) (uintptr, error) {
	return purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

// OpenLibrary opens a dynamic library with the platform dynamic loader.
func OpenLibrary(name string) (uintptr, error) {
	return loadLibrary(name)
}
