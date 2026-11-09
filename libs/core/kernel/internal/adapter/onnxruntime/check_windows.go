//go:build windows

package onnxruntime

import "syscall"

func checkLibrary(pos string) error {
	handle, err := syscall.LoadLibrary(pos)
	if err != nil {
		return err
	}
	symbol, err := syscall.GetProcAddress(handle, "OrtGetApiBase")
	if err != nil {
		return err
	}
	return checkAPI(symbol)
}
