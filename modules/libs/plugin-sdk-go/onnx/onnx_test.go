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

func TestResolveLibraryIsTheSettingOrElseTheOneTheHostGave(t *testing.T) {
	t.Setenv(EnvLibrary, "/host/libonnxruntime.so")
	if got := ResolveLibrary(""); got != "/host/libonnxruntime.so" {
		t.Errorf("ResolveLibrary() = %q, want the host's", got)
	}
	if got := ResolveLibrary("mine.so"); got != "mine.so" {
		t.Errorf("ResolveLibrary(mine.so) = %q, want the setting", got)
	}
}
