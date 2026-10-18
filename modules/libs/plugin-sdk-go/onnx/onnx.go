// Package onnx is the ONNX Runtime support plugins share: the library and the
// execution provider the host chose, the provider settings, and the one runtime
// a process opens. It links no binding; a plugin passes its own engine in.
package onnx

import "os"

const (
	// EnvLibrary names the path of the ONNX Runtime library.
	EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

	// EnvProvider names the execution provider to open sessions on: auto, cpu,
	// coreml, cuda, directml or rocm.
	EnvProvider = "NUMEN_ONNX_PROVIDER"
)

// DefaultProvider is the provider of a plugin the host gave none: it picks the
// best accelerator it can enable and falls back to the CPU.
const DefaultProvider = ProviderAuto

// GetLibrary returns the path of the library the host provisioned. It is empty
// for a plugin that does not declare the runtime.
func GetLibrary() string {
	return os.Getenv(EnvLibrary)
}

// ResolveLibrary returns the library a plugin's own setting names, or the one
// the host provisioned when the setting is empty.
func ResolveLibrary(configured string) string {
	if configured != "" {
		return configured
	}
	return GetLibrary()
}

// GetProvider returns the execution provider the host chose, or DefaultProvider.
func GetProvider() Provider {
	if name := os.Getenv(EnvProvider); name != "" {
		return Provider(name)
	}
	return DefaultProvider
}
