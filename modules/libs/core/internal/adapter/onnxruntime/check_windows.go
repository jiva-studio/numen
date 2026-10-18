//go:build windows

package onnxruntime

import "syscall"

// checkLibrary opens the library at a path with the platform loader and returns
// an error when it does not open or serves no apiVersion table. The library
// stays loaded.
func checkLibrary(at string) error {
	handle, err := syscall.LoadLibrary(at)
	if err != nil {
		return err
	}
	symbol, err := syscall.GetProcAddress(handle, "OrtGetApiBase")
	if err != nil {
		return err
	}
	return checkAPI(symbol)
}
