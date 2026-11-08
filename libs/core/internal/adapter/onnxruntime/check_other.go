//go:build !(darwin || freebsd || linux || netbsd || windows) || android

package onnxruntime

func checkLibrary(string) error {
	return nil
}
