// Package onnx holds what the engine plugins that run the ONNX Runtime share: the check, at the
// opening of a session, that a GPU provider asked for is one the library was built with.
package onnx

import (
	"fmt"
	"slices"
	"strings"
)

// EnvLibrary names the variable the host sets to the path of the library it provisioned, and the
// one a user sets to the library of a GPU build.
const EnvLibrary = "NUMEN_ONNX_RUNTIME_LIB"

// CheckProvider returns an error when provider names a GPU provider (cuda, rocm, directml) that the
// library at libPath was not built with. Another provider, and a platform whose library this package
// does not open, pass.
func CheckProvider(libPath, provider string) error {
	name, isGPU := getRuntimeName(provider)
	if !isGPU {
		return nil
	}
	available, err := listProviders(libPath)
	if err != nil {
		return fmt.Errorf("list the providers of the onnx runtime %s: %w", libPath, err)
	}
	return checkProviders(provider, name, libPath, available)
}

func getRuntimeName(provider string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "cuda":
		return "CUDAExecutionProvider", true
	case "rocm":
		return "ROCMExecutionProvider", true
	case "directml", "direct-ml", "dml":
		return "DmlExecutionProvider", true
	}
	return "", false
}

func checkProviders(provider, name, libPath string, available []string) error {
	if available == nil || slices.Contains(available, name) {
		return nil
	}
	return fmt.Errorf("the %s provider needs a GPU build of the onnx runtime, and %s is built with %s: install one and name it in %s, or use the cpu provider",
		provider, libPath, strings.Join(available, ", "), EnvLibrary)
}
