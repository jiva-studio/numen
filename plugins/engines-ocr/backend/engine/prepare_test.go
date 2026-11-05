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

func TestNewProbeConfig_SetsEveryPathToAMissingFileInTheProbeDirectory(t *testing.T) {
	probe := t.TempDir()
	cfg := newProbeConfig("/runtime/lib", probe)
	if cfg.OnnxRuntimeLibPath != "/runtime/lib" {
		t.Errorf("OnnxRuntimeLibPath = %q, want the library path", cfg.OnnxRuntimeLibPath)
	}
	for name, path := range map[string]string{"DetModelPath": cfg.DetModelPath, "RecModelPath": cfg.RecModelPath, "DictPath": cfg.DictPath} {
		if path == "" || filepath.Dir(path) != probe {
			t.Errorf("%s = %q, want a file in %q", name, path, probe)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s = %q, want a file that does not exist, stat error %v", name, path, err)
		}
	}
}
