package antigravity

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "agy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if got := New(nil).Find(""); got != filepath.Join(binDir, "agy") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "agy"), got)
	}
}

func TestDiscoveryWithoutAgents(t *testing.T) {
	withoutAgentsAnywhere(t)

	if got := findIn(nil); got != "agy" {
		t.Errorf("expected bare 'agy', got %q", got)
	}
}

func TestDiscoveryWithCustomPath(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "my-agy")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := New(nil).Find(custom); got != custom {
		t.Errorf("expected %s, got %s", custom, got)
	}
}

func TestACustomPathThatIsNotRunnableFallsBackToTheProgramFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
	binDir := t.TempDir()
	found := filepath.Join(binDir, "agy")
	if err := os.WriteFile(found, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if got := New(nil).Find(filepath.Join(t.TempDir(), "moved-agy")); got != found {
		t.Errorf("Find = %q, want %q", got, found)
	}
}

func TestDiscoveryPlacesAndFallbacks(t *testing.T) {
	if len(getPlaces()) == 0 {
		t.Fatal("expected non-empty getPlaces")
	}
	withoutAgentsAnywhere(t)

	if got := New(nil).Find(""); got != "" {
		t.Errorf("Find = %q, want none while absent", got)
	}
	if got := New(nil).Find("/nonexistent/agy-bin"); got != "" {
		t.Errorf("Find(nonexistent) = %q, want none", got)
	}
	if got := findIn(nil, ""); got != "agy" {
		t.Errorf("findIn(\"\", nil) = %q, want agy", got)
	}
	if got := findIn(nil, "/nonexistent/agy-bin"); got != "agy" {
		t.Errorf("findIn(nonexistent) = %q, want agy", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	inHome := filepath.Join(home, ".local", "bin", "agy")
	if err := os.MkdirAll(filepath.Dir(inHome), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inHome, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findIn([]string{"~/.local/bin/agy"}); got != inHome {
		t.Errorf("findIn(~/...) = %q, want %q", got, inHome)
	}

	// Empty HOME skips ~/ places and uses absolute place.
	absBin := filepath.Join(t.TempDir(), "abs-agy")
	if err := os.WriteFile(absBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", "")
	if got := findIn([]string{"~/.local/bin/agy", absBin}); got != absBin {
		t.Errorf("findIn with empty HOME = %q, want %q", got, absBin)
	}

}
