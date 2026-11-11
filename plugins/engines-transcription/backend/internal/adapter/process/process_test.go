package process

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine"
)

func TestGetConfigOverridesFromEnv(t *testing.T) {
	t.Setenv(engine.EnvDir, "")
	t.Setenv("NUMEN_ONNX_RUNTIME", "")
	base := Process{}.GetConfig()
	if base.Dir != "" || base.Runtime != "" {
		t.Fatalf("expected empty Dir and Runtime, got %+v", base)
	}

	t.Setenv(engine.EnvDir, "/env/models")
	t.Setenv("NUMEN_ONNX_RUNTIME", "/env/ort.so")
	fromEnv := Process{}.GetConfig()
	if fromEnv.Dir != "/env/models" || fromEnv.Runtime != "/env/ort.so" {
		t.Fatalf("expected env overrides, got %+v", fromEnv)
	}

}
