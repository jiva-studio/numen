//go:build darwin || freebsd || linux || netbsd

package onnx

import "github.com/ebitengine/purego"

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
