package antigravity

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	agenttest.NeedsShell(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "agy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if !isInstalled() {
		t.Error("expected Antigravity to be installed")
	}
	if got := findProgram(); got != filepath.Join(binDir, "agy") {
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
	if !isInstalled(custom) {
		t.Error("expected custom Antigravity to be installed")
	}
	if got := findProgram(custom); got != custom {
		t.Errorf("expected %s, got %s", custom, got)
	}
}

func TestDiscoveryPlacesAndFallbacks(t *testing.T) {
	if len(getPlaces()) == 0 {
		t.Fatal("expected non-empty getPlaces")
	}
	withoutAgentsAnywhere(t)

	if isInstalled("") {
		t.Error("expected empty customPath to report not installed when absent")
	}
	if isInstalled("/nonexistent/agy-bin") {
		t.Error("expected nonexistent customPath to report not installed")
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
