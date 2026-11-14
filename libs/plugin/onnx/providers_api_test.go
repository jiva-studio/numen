//go:build darwin || freebsd || linux || netbsd || windows

package onnx

import "testing"

func TestAGPUProviderOnALibraryThatDoesNotOpenFails(t *testing.T) {
	if err := CheckProvider("/no/such/libonnxruntime.so", "cuda"); err == nil {
		t.Error("a library that is not there passed the check")
	}
}
