package engine

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
)

func TestTheSessionsOpenOnTheCPUUnlessTheHostNamesAProvider(t *testing.T) {
	for name, want := range map[string]onnx.Provider{"": onnx.ProviderCPU, "auto": onnx.ProviderCPU, "cuda": onnx.ProviderCUDA} {
		t.Setenv("NUMEN_ONNX_PROVIDER", name)
		if got := onnx.GetSessionProvider(); got != want {
			t.Errorf("with %q, GetSessionProvider() = %q, want %q", name, got, want)
		}
	}
}

// A model opens with a language it transcribes, and not with another.
func TestAModelOpensOnlyWithALanguageItTranscribes(t *testing.T) {
	v2, _ := FindModel("parakeet-tdt-0.6b-v2-int8")
	for _, tc := range []struct {
		model    ParakeetModel
		language string
		isTaken  bool
	}{
		{Defaults().Model, "", true},
		{Defaults().Model, LanguageDetected, true},
		{Defaults().Model, "en", false},
		{v2, "en", true},
		{v2, LanguageDetected, false},
	} {
		cfg := Defaults()
		cfg.Model, cfg.Language = tc.model, tc.language
		if err := cfg.checkLanguage(); (err == nil) != tc.isTaken {
			t.Errorf("%s in %q: err = %v", tc.model.Name, tc.language, err)
		}
	}
}
