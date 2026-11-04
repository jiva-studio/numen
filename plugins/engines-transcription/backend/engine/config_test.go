package engine

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/engine/segmenter"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/onnx"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnx.Provider{"": onnx.ProviderCPU, "auto": onnx.ProviderCPU, "cuda": onnx.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := onnx.GetSessionProvider(); got != want {
			t.Errorf("with %q, GetSessionProvider() = %q, want %q", name, got, want)
		}
	}
}

func TestDefaultsAndGraphConstants(t *testing.T) {
	cfg := Defaults()
	if cfg.Runtime != "" || cfg.Dir != "" || cfg.Threads != 0 || cfg.Workers != 0 || cfg.Progress != nil || !cfg.ShouldDownload {
		t.Errorf("Defaults() = %+v", cfg)
	}
	if cfg.Model != getModels()[0] {
		t.Errorf("Defaults().Model = %+v, want %+v", cfg.Model, getModels()[0])
	}
	wantSegmenter := segmenter.Model{Name: "silero-vad", Repo: segmenterDefault}
	if cfg.Segmenter != wantSegmenter {
		t.Errorf("Defaults().Segmenter = %+v, want %+v", cfg.Segmenter, wantSegmenter)
	}
	if encoded != 1024 || hidden != 640 || layers != 2 {
		t.Errorf("graph constants = (%d, %d, %d), want (1024, 640, 2)", encoded, hidden, layers)
	}

	t.Setenv(onnx.EnvLibrary, "")
	if _, err := Open(t.Context(), NewRuntime(), Config{}); !errors.Is(err, onnx.ErrNoLibrary) {
		t.Errorf("Open without library = %v, want %v", err, onnx.ErrNoLibrary)
	}
}
