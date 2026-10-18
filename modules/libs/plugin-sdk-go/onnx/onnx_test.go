package onnx

import (
	"os"
	"path/filepath"
	"testing"
)

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
		if got := GetProviderOr(ProviderCPU); got != ProviderCPU {
			t.Errorf("with %q, GetProviderOr(cpu) = %q, want cpu", name, got)
		}
	}
	t.Setenv(EnvProvider, "cuda")
	if got := GetProviderOr(ProviderCPU); got != ProviderCUDA {
		t.Errorf("GetProviderOr(cpu) = %q, want the host's cuda", got)
	}
}

func TestResolveLibraryIsTheSettingThenTheFolderThenTheHost(t *testing.T) {
	t.Setenv(EnvLibrary, "/host/libonnxruntime.so")
	dir := t.TempDir()
	at := filepath.Join(dir, GetLibraryName())
	if err := os.WriteFile(at, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, one := range []struct{ configured, dir, want string }{
		{"mine.so", dir, "mine.so"},
		{"", dir, at},
		{"", t.TempDir(), "/host/libonnxruntime.so"},
		{"", "", "/host/libonnxruntime.so"},
	} {
		if got := ResolveLibrary(one.configured, one.dir); got != one.want {
			t.Errorf("ResolveLibrary(%q, %q) = %q, want %q", one.configured, one.dir, got, one.want)
		}
	}
}
