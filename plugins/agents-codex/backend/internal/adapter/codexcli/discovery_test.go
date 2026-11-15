package codexcli

import (
	"os"
	"path/filepath"
	"runtime"
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

	// Empty HOME does not expand ~/ and falls through to absolute place, even when relative .local/bin/codex exists in CWD.
	absBin := filepath.Join(t.TempDir(), "abs-codex")
	if err := os.WriteFile(absBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findCodexIn([]string{absBin}); got != absBin {
		t.Errorf("findCodexIn(absBin) with HOME set = %q, want %q", got, absBin)
	}
	workDir := t.TempDir()
	relBin := filepath.Join(workDir, ".local", "bin", "codex")
	if err := os.MkdirAll(filepath.Dir(relBin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(workDir)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if got := findCodexIn([]string{"~/.local/bin/codex", absBin}); got != absBin {
		t.Errorf("findCodexIn with empty HOME = %q, want %q", got, absBin)
	}

	if isRunnable(t.TempDir()) {
		t.Error("a directory should not be runnable")
	}
	if runtime.GOOS != "windows" {
		modeFile := filepath.Join(t.TempDir(), "mode-check")
		if err := os.WriteFile(modeFile, []byte("#!/bin/sh\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			mode os.FileMode
			want bool
		}{
			{0o600, false},
			{0o002, false},
			{0o001, true},
			{0o010, true},
			{0o100, true},
		} {
			if err := os.Chmod(modeFile, tc.mode); err != nil {
				t.Fatal(err)
			}
			if got := isRunnable(modeFile); got != tc.want {
				t.Errorf("isRunnable(mode %04o) = %v, want %v", tc.mode, got, tc.want)
			}
		}
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
