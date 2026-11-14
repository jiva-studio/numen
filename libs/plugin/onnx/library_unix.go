//go:build darwin || freebsd || linux || netbsd

package onnx

import "github.com/ebitengine/purego"

func openLibrary(libPath string) (uintptr, error) {
	return purego.Dlopen(libPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}

func findSymbol(handle uintptr, name string) (uintptr, error) {
	return purego.Dlsym(handle, name)
}
