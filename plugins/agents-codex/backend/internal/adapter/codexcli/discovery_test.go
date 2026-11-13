package codexcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	needsShell(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if got := (Installation{}).Find(""); got != filepath.Join(binDir, "codex") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "codex"), got)
	}
}

func TestDiscoveryWithoutAgents(t *testing.T) {
	withoutAgentsAnywhere(t)

	if got := (Installation{}).Find(""); got != "" {
		t.Errorf("Find = %q, want none", got)
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
	if got := (Installation{}).Find(custom); got != custom {
		t.Errorf("expected %s, got %s", custom, got)
	}
}

func TestACustomPathThatIsNotRunnableFallsBackToTheCodexFound(t *testing.T) {
	needsShell(t)
	binDir := t.TempDir()
	found := filepath.Join(binDir, "codex")
	if err := os.WriteFile(found, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	if got := (Installation{}).Find(filepath.Join(t.TempDir(), "moved-codex")); got != found {
		t.Errorf("Find = %q, want %q", got, found)
	}
}

func TestDiscoveryPlacesAndFallbacks(t *testing.T) {
	if len(getCodexPlaces()) == 0 {
		t.Fatal("expected non-empty getCodexPlaces")
	}
	withoutAgentsAnywhere(t)

	if got := (Installation{}).Find("/nonexistent/codex-bin"); got != "" {
		t.Errorf("Find(nonexistent) = %q, want none", got)
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
