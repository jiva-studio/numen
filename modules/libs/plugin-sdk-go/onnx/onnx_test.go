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

func TestNothingIsSetForAPluginTheHostGaveNoRuntime(t *testing.T) {
	t.Setenv(EnvLibrary, "")
	t.Setenv(EnvProvider, "")
	if GetLibrary() != "" || GetProvider() != "" {
		t.Errorf("GetLibrary() = %q, GetProvider() = %q", GetLibrary(), GetProvider())
	}
}
