package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/onnx"
)

func TestResolveLibrary_ModelsDirPrecedesHostRuntime(t *testing.T) {
	t.Setenv(onnx.EnvLibrary, "/host/lib")
	models := t.TempDir()
	library := filepath.Join(models, onnx.GetLibraryName())
	if err := os.WriteFile(library, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := onnx.ResolveLibrary((settings.Config{Dir: models}).Runtime, (settings.Config{Dir: models}).Dir); got != library {
		t.Errorf("with a library in dir, runtime = %q", got)
	}
	if got := onnx.ResolveLibrary((settings.Config{Dir: models, Runtime: "set.so"}).Runtime, (settings.Config{Dir: models, Runtime: "set.so"}).Dir); got != "set.so" {
		t.Errorf("with a runtime setting, runtime = %q", got)
	}
}
