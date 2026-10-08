//go:build !(darwin || freebsd || linux || netbsd)

package onnxruntime

import "syscall"

// support is nothing where the runtime carries what it needs beside it.
func support() {}

func loadLibrary(name string) (uintptr, error) {
	h, err := syscall.LoadLibrary(name)
	return uintptr(h), err
}
