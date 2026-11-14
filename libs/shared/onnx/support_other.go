//go:build !(darwin || freebsd || linux || netbsd)

package onnx

import "syscall"

func preloadCppRuntime() {}

func loadLibrary(name string) (uintptr, error) {
	h, err := syscall.LoadLibrary(name)
	return uintptr(h), err
}

// OpenLibrary opens a dynamic library with the platform dynamic loader.
func OpenLibrary(name string) (uintptr, error) {
	return loadLibrary(name)
}
