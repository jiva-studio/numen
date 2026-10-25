package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

func TestTheRuntimeInTheModelsFolderReachesBeforeTheOneTheHostGave(t *testing.T) {
	t.Setenv("NUMEN_ONNX_RUNTIME_LIB", "/host/libonnxruntime.so")
	dir := t.TempDir()
	pos := filepath.Join(dir, onnx.GetLibraryName())
	if err := os.WriteFile(pos, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []struct {
		Config
		want string
	}{{Config{Dir: dir}, pos}, {Config{Dir: dir, Runtime: "mine.so"}, "mine.so"}, {Config{Dir: t.TempDir()}, "/host/libonnxruntime.so"}} {
		if got := onnx.ResolveLibrary(cfg.Runtime, cfg.Dir); got != cfg.want {
			t.Errorf("%+v: runtime = %q, want %q", cfg.Config, got, cfg.want)
		}
	}
}
