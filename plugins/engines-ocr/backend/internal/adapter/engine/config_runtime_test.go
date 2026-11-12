package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/engine/onnx"
)

func TestResolveLibrary_OrdersSettingDirThenHost(t *testing.T) {
	t.Setenv("NUMEN_ONNX_RUNTIME_LIB", "/host/libonnxruntime.so")
	dir := t.TempDir()
	pos := filepath.Join(dir, onnx.GetLibraryName())
	if err := os.WriteFile(pos, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []struct {
		settings.Config
		want string
	}{{settings.Config{Dir: dir}, pos}, {settings.Config{Dir: dir, Runtime: "mine.so"}, "mine.so"}, {settings.Config{Dir: t.TempDir()}, "/host/libonnxruntime.so"}} {
		if got := onnx.ResolveLibrary(cfg.Runtime, cfg.Dir); got != cfg.want {
			t.Errorf("%+v: runtime = %q, want %q", cfg.Config, got, cfg.want)
		}
	}
}
