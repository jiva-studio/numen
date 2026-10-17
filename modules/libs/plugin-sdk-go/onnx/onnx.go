// Package onnx is what the host tells a plugin that declares the onnxruntime
// runtime: the library it provisioned and the execution provider it chose.
package onnx

import "os"

const (
	// EnvLibrary names the absolute path of the ONNX Runtime library.
	EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

	// EnvProvider names the execution provider to open sessions on: cpu or coreml.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// GetLibrary returns the path of the library the host provisioned. It is empty
// for a plugin that does not declare the runtime.
func GetLibrary() string {
	return os.Getenv(EnvLibrary)
}

// GetProvider returns the execution provider the host chose. It is empty for a
// plugin that does not declare the runtime.
func GetProvider() string {
	return os.Getenv(EnvProvider)
}
