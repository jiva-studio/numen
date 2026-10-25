package onnxruntime

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEveryPublishedReleaseIsPinned(t *testing.T) {
	if len(releases) == 0 {
		t.Fatal("no release is published")
	}
	for platform, release := range releases {
		if err := release.check(); err != nil {
			t.Errorf("%s: %v", platform, err)
		}
	}
}

func TestNoReleaseIsPublishedForAPlatformThatHasNone(t *testing.T) {
	if _, err := getRelease("plan9", "mips"); err == nil {
		t.Fatal("getRelease() found a release for plan9/mips")
	}
}

func TestParseProvider(t *testing.T) {
	names := map[string]Provider{
		"": ProviderAuto, "auto": ProviderAuto, "cpu": ProviderCPU, "coreml": ProviderCoreML, "core-ml": ProviderCoreML,
		"cuda": ProviderCUDA, "directml": ProviderDirectML, "dml": ProviderDirectML, "rocm": ProviderROCm,
	}
	for name, want := range names {
		if got, err := ParseProvider(name); err != nil || got != want {
			t.Errorf("ParseProvider(%q) = %q, %v, want %q", name, got, err, want)
		}
	}
	if _, err := ParseProvider("tensorrt"); err == nil {
		t.Error("ParseProvider() accepted tensorrt")
	}
}

func TestOnlyTheGPUProvidersNeedAGPUBuild(t *testing.T) {
	for provider, want := range map[Provider]bool{
		ProviderAuto: false, ProviderCPU: false, ProviderCoreML: false,
		ProviderCUDA: true, ProviderDirectML: true, ProviderROCm: true,
	} {
		if got := provider.isGPU(); got != want {
			t.Errorf("%s.isGPU() = %v, want %v", provider, got, want)
		}
	}
}

func TestEachPlatformNamesItsLibrary(t *testing.T) {
	for goos, want := range map[string]string{"linux": "libonnxruntime.so", "darwin": "libonnxruntime.dylib", "windows": "onnxruntime.dll"} {
		if got := getLibraryName(goos); got != want {
			t.Errorf("getLibraryName(%q) = %q, want %q", goos, got, want)
		}
	}
}

func TestAnExplicitMachineListIsTheOnlyPlaceSearched(t *testing.T) {
	fakeNix := t.TempDir()
	libDir := filepath.Join(fakeNix, "pkg-onnxruntime-1", "lib")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	libPath := filepath.Join(libDir, getLibraryName(runtime.GOOS))
	if err := os.WriteFile(libPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	testPattern := filepath.Join(fakeNix, "*-onnxruntime-*", "lib")
	if pos := getCandidatesInStore(t.TempDir(), "", "", []string{}, testPattern); len(pos) != 0 {
		t.Errorf("getCandidatesInStore() = %q with explicit empty machine folders, wanted empty", pos)
	}
	emptyMachine := []string{t.TempDir()}
	if offset := getCandidatesInStore(t.TempDir(), "", "", emptyMachine, testPattern); len(offset) != 0 {
		t.Errorf("getCandidatesInStore() = %q with explicit machine folders, wanted empty", offset)
	}
	fakeMachine := []string{t.TempDir()}
	if index := getCandidatesInStore(t.TempDir(), "", "", fakeMachine, testPattern); len(index) != 0 {
		t.Errorf("getCandidatesInStore() = %q with fake machine folders, wanted empty", index)
	}
}
