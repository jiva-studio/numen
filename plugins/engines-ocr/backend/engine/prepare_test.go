package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/onnx"
)

// findRuntime returns an ONNX Runtime library present on this machine: the one the host names, else
// one provisioned in the user's Numen models folder.
func findRuntime(t *testing.T) string {
	t.Helper()
	if lib := onnx.GetLibrary(); lib != "" {
		return lib
	}
	if base, err := os.UserConfigDir(); err == nil {
		pattern := filepath.Join(base, "numen", "plugin-storage", "shared", "models", "onnxruntime", "*", onnx.GetLibraryName())
		if found, _ := filepath.Glob(pattern); len(found) > 0 {
			return found[0]
		}
	}
	return ""
}

func TestBindRuntime_BindsARealRuntime(t *testing.T) {
	lib := findRuntime(t)
	if lib == "" {
		t.Skipf("no %s found: set %s or provision the runtime in the Numen models folder", onnx.GetLibraryName(), onnx.EnvLibrary)
	}
	if err := bindRuntime(lib); err != nil {
		t.Fatalf("bindRuntime(%q) = %v, want the runtime bound", lib, err)
	}
}
