package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/onnx"
)

func TestARuntimeBesideTheModelsIsTakenAheadOfTheHostsAndBehindTheSetting(t *testing.T) {
	t.Setenv(onnx.EnvLibrary, "/host/lib")
	models := t.TempDir()
	library := filepath.Join(models, onnx.GetLibraryName())
	if err := os.WriteFile(library, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := (Config{Dir: models}).settings().Runtime; got != library {
		t.Errorf("with a library in dir, runtime = %q", got)
	}
	if got := (Config{Dir: models, Runtime: "set.so"}).settings().Runtime; got != "set.so" {
		t.Errorf("with a runtime setting, runtime = %q", got)
	}
}
