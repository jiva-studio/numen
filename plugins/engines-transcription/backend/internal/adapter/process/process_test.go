package process

import (
	"testing"
)

func TestGetConfigOverridesFromEnv(t *testing.T) {
	t.Setenv(envModelsDir, "")
	t.Setenv("NUMEN_ONNX_RUNTIME", "")
	base := Process{}.GetConfig()
	if base.Dir != "" || base.Runtime != "" {
		t.Fatalf("expected empty Dir and Runtime, got %+v", base)
	}

	t.Setenv(envModelsDir, "/env/models")
	t.Setenv("NUMEN_ONNX_RUNTIME", "/env/ort.so")
	fromEnv := Process{}.GetConfig()
	if fromEnv.Dir != "/env/models" || fromEnv.Runtime != "/env/ort.so" {
		t.Fatalf("expected env overrides, got %+v", fromEnv)
	}

}
