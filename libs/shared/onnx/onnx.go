// Package onnx is the ONNX Runtime support plugins share: the library and the
// execution provider the host chose, the provider settings, and the one runtime
// a process opens through the onnxruntime_purego binding, with the sessions
// opened on it.
package onnx

import (
	"cmp"
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
const DefaultProvider = ProviderAuto

// GetLibrary returns the path of the library the host provisioned. It is empty
// for a plugin that does not declare the runtime.
func GetLibrary() string {
	return os.Getenv(EnvLibrary)
}

// ResolveLibrary returns the library a plugin opens: the one its own setting
// names, else the one in the models folder its settings name, else the one the
// host provisioned.
func ResolveLibrary(configured, dir string) string {
	return cmp.Or(configured, FindLibrary(dir), GetLibrary())
}

// GetProvider returns the execution provider the host chose, or DefaultProvider.
func GetProvider(configured string) Provider {
	if name := os.Getenv(EnvProvider); name != "" {
		return Provider(name)
	}
	return Provider(cmp.Or(configured, string(DefaultProvider)))
}

// GetSessionProvider returns the execution provider a plugin opens its sessions
// on: the one the host named, else the CPU.
func GetSessionProvider(configured string) Provider {
	if name := GetProvider(configured); !strings.EqualFold(strings.TrimSpace(string(name)), string(DefaultProvider)) {
		return name
	}
	return ProviderCPU
}

// LimitThreads bounds the requested session thread count by the host budget. A zero limit is unconstrained.
func LimitThreads(requested, limit int) int {
	if limit > 0 {
		return min(requested, limit)
	}
	return requested
}
