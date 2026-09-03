//go:build !(darwin || freebsd || linux || netbsd)

package onnxruntime

// support is nothing where the runtime carries what it needs beside it.
func support() {}
