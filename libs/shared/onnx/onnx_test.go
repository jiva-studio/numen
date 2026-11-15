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

func TestGetProvider_DefaultsToAutoWithoutHostRuntime(t *testing.T) {
	t.Setenv(EnvLibrary, "")
	t.Setenv(EnvProvider, "")
	if GetLibrary() != "" || GetProvider() != DefaultProvider {
		t.Errorf("GetLibrary() = %q, GetProvider() = %q", GetLibrary(), GetProvider())
	}
}

func TestResolveLibraryIsTheSettingThenTheFolderThenTheHost(t *testing.T) {
	t.Setenv(EnvLibrary, "/host/libonnxruntime.so")
	dir := t.TempDir()
	targetPath := filepath.Join(dir, GetLibraryName())
	if err := os.WriteFile(targetPath, []byte("a library"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, one := range []struct{ configured, dir, want string }{
		{"mine.so", dir, "mine.so"},
		{"", dir, targetPath},
		{"", t.TempDir(), "/host/libonnxruntime.so"},
		{"", "", "/host/libonnxruntime.so"},
	} {
		if got := ResolveLibrary(one.configured, one.dir); got != one.want {
			t.Errorf("ResolveLibrary(%q, %q) = %q, want %q", one.configured, one.dir, got, one.want)
		}
	}
}
