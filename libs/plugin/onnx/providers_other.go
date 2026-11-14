//go:build !(darwin || freebsd || linux || netbsd || windows)

package onnx

func listProviders(string) ([]string, error) {
	return nil, nil
}
