package onnx

import "testing"

func TestGetLibraryAndGetProviderAreWhatTheHostSet(t *testing.T) {
	t.Setenv(EnvLibrary, "/models/libonnxruntime.so")
	t.Setenv(EnvProvider, "coreml")
	if got := GetLibrary(); got != "/models/libonnxruntime.so" {
		t.Errorf("GetLibrary() = %q", got)
	}
	if got := GetProvider(); got != "coreml" {
		t.Errorf("GetProvider() = %q", got)
	}
}

func TestAPluginTheHostGaveNoRuntimeHasNoLibraryAndTheAutoProvider(t *testing.T) {
	t.Setenv(EnvLibrary, "")
	t.Setenv(EnvProvider, "")
	if GetLibrary() != "" || GetProvider() != DefaultProvider {
		t.Errorf("GetLibrary() = %q, GetProvider() = %q", GetLibrary(), GetProvider())
	}
}

func TestGetProviderOrIsTheFallbackForAuto(t *testing.T) {
	for _, name := range []string{"", "auto", " AUTO "} {
		t.Setenv(EnvProvider, name)
		if got := GetProviderOr("cpu"); got != "cpu" {
			t.Errorf("with %q, GetProviderOr(cpu) = %q, want cpu", name, got)
		}
	}
	t.Setenv(EnvProvider, "cuda")
	if got := GetProviderOr("cpu"); got != "cuda" {
		t.Errorf("GetProviderOr(cpu) = %q, want the host's cuda", got)
	}
}
