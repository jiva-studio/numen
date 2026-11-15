//go:build !(darwin || freebsd || linux || netbsd)

package onnx

import "syscall"

func preloadCppRuntime() {}

// OpenLibrary opens a dynamic library with the platform dynamic loader.
func OpenLibrary(name string) (uintptr, error) {
	h, err := syscall.LoadLibrary(name)
	return uintptr(h), err
}

func findSymbol(handle uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(handle), name)
}
