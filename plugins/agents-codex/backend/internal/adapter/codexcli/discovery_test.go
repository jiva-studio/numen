package codexcli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun/agenttest"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	agenttest.NeedsShell(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if !IsCodexInstalled() {
		t.Error("expected Codex to be installed")
	}
	if got := FindCodex(); got != filepath.Join(binDir, "codex") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "codex"), got)
	}
}

func TestDiscoveryWithoutAgents(t *testing.T) {
	withoutAgentsAnywhere(t)

	if IsCodexInstalled() {
		t.Error("expected Codex to be absent")
	}
	if got := findCodexIn(nil); got != "codex" {
		t.Errorf("expected bare 'codex', got %q", got)
	}
}

func TestDiscoveryWithCustomPath(t *testing.T) {
	tmp := t.TempDir()
	custom := filepath.Join(tmp, "my-codex")
	if err := os.WriteFile(custom, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsCodexInstalled(custom) {
		t.Error("expected custom Codex to be installed")
	}
	if got := FindCodex(custom); got != custom {
		t.Errorf("expected %s, got %s", custom, got)
	}
}

func TestDiscoveryPlacesAndFallbacks(t *testing.T) {
	if len(getCodexPlaces()) == 0 {
		t.Fatal("expected non-empty getCodexPlaces")
	}
	withoutAgentsAnywhere(t)

	if IsCodexInstalled("") {
		t.Error("expected empty customPath to fall through and report not installed")
	}
	if IsCodexInstalled("/nonexistent/codex-bin") {
		t.Error("expected nonexistent customPath to report not installed")
	}
	if got := findCodexIn(nil, ""); got != "codex" {
		t.Errorf("findCodexIn(\"\", nil) = %q, want codex", got)
	}
	if got := findCodexIn(nil, "/nonexistent/codex-bin"); got != "codex" {
		t.Errorf("findCodexIn(nonexistent) = %q, want codex", got)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	inHome := filepath.Join(home, ".local", "bin", "codex")
	if err := os.MkdirAll(filepath.Dir(inHome), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inHome, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findCodexIn([]string{"~/.local/bin/codex"}); got != inHome {
		t.Errorf("findCodexIn(~/...) = %q, want %q", got, inHome)
	}

	// Empty HOME does not expand ~/ and falls through to absolute place.
	absBin := filepath.Join(t.TempDir(), "abs-codex")
	if err := os.WriteFile(absBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", "")
	if got := findCodexIn([]string{"~/.local/bin/codex", absBin}); got != absBin {
		t.Errorf("findCodexIn with empty HOME = %q, want %q", got, absBin)
	}
}

func withoutAgentsAnywhere(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if found := FindCodex(); found != "codex" {
		t.Skipf("Codex is installed at %s", found)
	}
}
