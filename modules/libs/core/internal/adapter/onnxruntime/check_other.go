//go:build !(darwin || freebsd || linux || netbsd || windows) || android

package onnxruntime

// checkLibrary accepts every library on a platform with no loader to open one
// through.
func checkLibrary(string) error {
	return nil
}
