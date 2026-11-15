package engine

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/onnx"
)

func findRuntime(t *testing.T) string {
	t.Helper()
	if lib := onnx.GetLibrary(); lib != "" {
		return lib
	}
	if base, err := os.UserConfigDir(); err == nil {
		pattern := filepath.Join(base, "numen", "plugin-storage", "shared", "models", "onnxruntime", "*", onnx.GetLibraryName())
		found, globErr := filepath.Glob(pattern)
		if globErr != nil {
			t.Fatalf("glob %q: %v", pattern, globErr)
		}
		if len(found) > 0 {
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
	if cfg.UseCuda || cfg.NumThreads != 0 || cfg.ThreadCount != 0 || cfg.DetMaxSideLen != 0 ||
		cfg.DetOutsideExpandPix != 0 || cfg.RecHeight != 0 || cfg.RecModelNumClasses != 0 || cfg.HeatmapThreshold != 0 {
		t.Errorf("probe numeric/boolean fields = %+v, want all zero/false", cfg)
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

func TestBindRuntimeAndCheckBindError(t *testing.T) {
	if err := checkBindError("/lib", nil); err != nil {
		t.Fatalf("checkBindError(nil) = %v, want nil", err)
	}
	if err := checkBindError("/lib", fs.ErrNotExist); err != nil {
		t.Fatalf("checkBindError(ErrNotExist) = %v, want nil", err)
	}
	errBoom := errors.New("dlopen failed")
	if err := checkBindError("/lib", errBoom); !errors.Is(err, errBoom) || !strings.Contains(err.Error(), "bind the runtime at /lib") {
		t.Fatalf("checkBindError(errBoom) = %v, want wrapped errBoom", err)
	}

	missingLib := filepath.Join(t.TempDir(), "missing-libonnxruntime.so")
	if err := bindRuntime(missingLib); err == nil || !strings.Contains(err.Error(), "bind the runtime at "+missingLib) {
		t.Fatalf("bindRuntime(missingLib) = %v, want bind error", err)
	}

	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "no-such-tmp"))
	if err := bindRuntime("/lib"); err == nil || !strings.Contains(err.Error(), "create the probe directory") {
		t.Fatalf("bindRuntime with invalid TMPDIR = %v, want probe directory error", err)
	}
}
