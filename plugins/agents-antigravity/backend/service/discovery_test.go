package service

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	agenttest.NeedsShell(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "agy"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if !IsAntigravityInstalled() {
		t.Error("expected Antigravity to be installed")
	}
	if got := FindAntigravity(); got != filepath.Join(binDir, "agy") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "agy"), got)
	}
}

func TestDiscoveryWithoutAgents(t *testing.T) {
	withoutAgentsAnywhere(t)

	if got := findAntigravityIn(nil); got != "agy" {
		t.Errorf("expected bare 'agy', got %q", got)
	}
}

func TestDiscoveryWithCustomPath(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "my-agy")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsAntigravityInstalled(custom) {
		t.Error("expected custom Antigravity to be installed")
	}
	if got := FindAntigravity(custom); got != custom {
		t.Errorf("expected %s, got %s", custom, got)
	}
}

func TestDiscoveryPlacesAndFallbacks(t *testing.T) {
	if len(getAntigravityPlaces()) == 0 {
		t.Fatal("expected non-empty getAntigravityPlaces")
	}
	withoutAgentsAnywhere(t)

	if IsAntigravityInstalled("") {
		t.Error("expected empty customPath to report not installed when absent")
	}
	if IsAntigravityInstalled("/nonexistent/agy-bin") {
		t.Error("expected nonexistent customPath to report not installed")
	}
	if got := findAntigravityIn(nil, ""); got != "agy" {
		t.Errorf("findAntigravityIn(\"\", nil) = %q, want agy", got)
	}
	if got := findAntigravityIn(nil, "/nonexistent/agy-bin"); got != "agy" {
		t.Errorf("findAntigravityIn(nonexistent) = %q, want agy", got)
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
	if got := findAntigravityIn([]string{"~/.local/bin/agy"}); got != inHome {
		t.Errorf("findAntigravityIn(~/...) = %q, want %q", got, inHome)
	}

	// Empty HOME skips ~/ places and uses absolute place.
	absBin := filepath.Join(t.TempDir(), "abs-agy")
	if err := os.WriteFile(absBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", "")
	if got := findAntigravityIn([]string{"~/.local/bin/agy", absBin}); got != absBin {
		t.Errorf("findAntigravityIn with empty HOME = %q, want %q", got, absBin)
	}

	cmds := []agentrun.ProgramCommand{
		{Program: "other", Command: []string{"other-bin"}},
		{Program: "antigravity", Command: []string{"agy-bin", "--arg"}},
	}
	if got := programCommand(cmds, "antigravity"); !slices.Equal(got, []string{"agy-bin", "--arg"}) {
		t.Errorf("programCommand(antigravity) = %v", got)
	}
	if got := programCommand(cmds, "missing"); got != nil {
		t.Errorf("programCommand(missing) = %v, want nil", got)
	}
}
