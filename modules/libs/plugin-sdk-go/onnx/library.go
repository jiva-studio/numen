package onnx

import (
	"os"
	"path/filepath"
	"runtime"
)

// FindLibrary returns the path of the ONNX Runtime library that stands in dir,
// or empty when dir is empty or holds none.
func FindLibrary(dir string) string {
	if dir == "" {
		return ""
	}
	at := filepath.Join(dir, GetLibraryName())
	if info, err := os.Stat(at); err == nil && info.Mode().IsRegular() {
		return at
	}
	return ""
}

// GetLibraryName returns what the library is called on this operating system.
func GetLibraryName() string {
	return getLibraryName(runtime.GOOS)
}

// getLibraryName returns what the library is called on an operating system.
func getLibraryName(goos string) string {
	switch goos {
	case "darwin":
		return "libonnxruntime.dylib"
	case "windows":
		return "onnxruntime.dll"
	}
	return "libonnxruntime.so"
}
