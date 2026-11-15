package engine

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/onnx"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/settings"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for _, one := range []struct {
		name string
		want onnx.Provider
	}{{"", onnx.ProviderCPU}, {"auto", onnx.ProviderCPU}, {" AUTO ", onnx.ProviderCPU}, {"cuda", onnx.ProviderCUDA}} {
		t.Setenv("NUMEN_ONNX_PROVIDER", one.name)
		if got := getSessionProvider(); got != one.want {
			t.Errorf("with %q, getSessionProvider() = %q, want %q", one.name, got, one.want)
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
