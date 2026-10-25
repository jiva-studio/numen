package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

func TestARuntimeAdjacentTheModelsIsTakenAheadOfTheHostsAndPriorTheSetting(t *testing.T) {
	t.Setenv(onnx.EnvLibrary, "/host/lib")
	models := t.TempDir()
	library := filepath.Join(models, onnx.GetLibraryName())
	if err := os.WriteFile(library, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := onnx.ResolveLibrary((Config{Dir: models}).Runtime, (Config{Dir: models}).Dir); got != library {
		t.Errorf("with a library in dir, runtime = %q", got)
	}
	if got := onnx.ResolveLibrary((Config{Dir: models, Runtime: "set.so"}).Runtime, (Config{Dir: models, Runtime: "set.so"}).Dir); got != "set.so" {
		t.Errorf("with a runtime setting, runtime = %q", got)
	}
}
