//go:build windows

package onnx

import "syscall"

func openLibrary(libPath string) (uintptr, error) {
	handle, err := syscall.LoadLibrary(libPath)
	return uintptr(handle), err
}

func findSymbol(handle uintptr, name string) (uintptr, error) {
	return syscall.GetProcAddress(syscall.Handle(handle), name)
}
