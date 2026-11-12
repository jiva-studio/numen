package engine

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/internal/adapter/engine/onnx"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnx.Provider{"": onnx.ProviderCPU, "auto": onnx.ProviderCPU, "cuda": onnx.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := onnx.GetSessionProvider(); got != want {
			t.Errorf("with %q, GetSessionProvider() = %q, want %q", name, got, want)
		}
	}
}

func TestGraphConstants(t *testing.T) {
	if encoded != 1024 || hidden != 640 || layers != 2 {
		t.Errorf("graph constants = (%d, %d, %d), want (1024, 640, 2)", encoded, hidden, layers)
	}

	t.Setenv(onnx.EnvLibrary, "")
	if _, err := Open(t.Context(), NewRuntime(), settings.Config{}); !errors.Is(err, onnx.ErrNoLibrary) {
		t.Errorf("Open without library = %v, want %v", err, onnx.ErrNoLibrary)
	}
}
