// Package onnx is what the host tells a plugin that declares the onnxruntime
// runtime: the library it provisioned and the execution provider it chose.
package onnx

import (
	"os"
	"strings"
)

const (
	// EnvLibrary names the path of the ONNX Runtime library.
	EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

	// EnvProvider names the execution provider to open sessions on: auto, cpu,
	// coreml, cuda, directml or rocm.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// DefaultProvider is the provider of a plugin the host gave none: it picks the
// best accelerator it can enable and falls back to the CPU.
const DefaultProvider = "auto"

// GetLibrary returns the path of the library the host provisioned. It is empty
// for a plugin that does not declare the runtime.
func GetLibrary() string {
	return os.Getenv(EnvLibrary)
}

// GetProvider returns the execution provider the host chose, or DefaultProvider.
func GetProvider() string {
	if name := os.Getenv(EnvProvider); name != "" {
		return name
	}
	return DefaultProvider
}

// GetProviderOr returns the execution provider the host chose, or fallback when
// it chose auto or none.
func GetProviderOr(fallback string) string {
	if name := GetProvider(); !strings.EqualFold(strings.TrimSpace(name), DefaultProvider) {
		return name
	}
	return fallback
}
