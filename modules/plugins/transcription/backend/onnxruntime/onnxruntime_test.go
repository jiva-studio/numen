package onnxruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenNamesTheVariableWhenTheHostGaveNoLibrary(t *testing.T) {
	if _, _, err := Open(""); err == nil || !strings.Contains(err.Error(), EnvLibrary) {
		t.Fatalf("Open() = %v, want %s named", err, EnvLibrary)
	}
}

func TestOpenRefusesALibraryThatIsNotThere(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if _, _, err := Open(missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("Open() = %v, want %s named", err, missing)
	}
}

func TestGetLibraryIsWhatTheHostSet(t *testing.T) {
	t.Setenv(EnvLibrary, "/models/libonnxruntime.so")
	if got := GetLibrary(); got != "/models/libonnxruntime.so" {
		t.Errorf("GetLibrary() = %q", got)
	}
}

func TestALibraryIsAvailableWhereTheFileIs(t *testing.T) {
	if IsAvailable("") {
		t.Error("an empty path is available")
	}
	at := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if IsAvailable(at) {
		t.Error("a missing file is available")
	}
	if err := os.WriteFile(at, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !IsAvailable(at) {
		t.Error("a file that is there is not available")
	}
}

func TestSessionsOpenOnTheProviderTheHostChose(t *testing.T) {
	t.Setenv(EnvProvider, "cuda")
	if got, err := resolveProvider(""); err != nil || got != ProviderCUDA {
		t.Errorf("resolveProvider() = %q, %v, want cuda", got, err)
	}
	if got, err := resolveProvider(ProviderCPU); err != nil || got != ProviderCPU {
		t.Errorf("resolveProvider(cpu) = %q, %v, want cpu", got, err)
	}
	t.Setenv(EnvProvider, "tensorrt")
	if _, err := resolveProvider(""); err == nil {
		t.Error("resolveProvider() accepted a provider nothing here opens")
	}
	t.Setenv(EnvProvider, "")
	if got, err := resolveProvider(""); err != nil || got != ProviderAuto {
		t.Errorf("resolveProvider() = %q, %v, want auto", got, err)
	}
}

func TestParseProviderAcceptsWhatTheHostChooses(t *testing.T) {
	for name, want := range map[string]Provider{"cpu": ProviderCPU, "cuda": ProviderCUDA, "coreml": ProviderCoreML, "": ProviderAuto} {
		if got, err := ParseProvider(name); err != nil || got != want {
			t.Errorf("ParseProvider(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
	if _, err := ParseProvider("directml"); err == nil {
		t.Error("ParseProvider() accepted a provider the host never chooses")
	}
}
