package agenttest

import (
	"os"
	"path/filepath"
	"testing"
)

// CheckDiscovery fails where the discovery functions of the program installed as binary do not find
// it on PATH, do not return the bare name where it is absent, or do not prefer a runnable custom path.
// withoutAgents makes the machine hold none of the programs, wherever they are searched for.
func CheckDiscovery(
	t *testing.T,
	binary string,
	find func(customPath ...string) string,
	isInstalled func(customPath ...string) bool,
	withoutAgents func(t *testing.T),
) {
	t.Helper()
	t.Run("finds the program on PATH", func(t *testing.T) {
		binDir := t.TempDir()
		writeProgram(t, filepath.Join(binDir, binary))
		t.Setenv("PATH", binDir)

		if !isInstalled() {
			t.Errorf("expected %s to be installed", binary)
		}
		if got, want := find(), filepath.Join(binDir, binary); got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	})
	t.Run("returns the bare name where the program is absent", func(t *testing.T) {
		withoutAgents(t)

		if isInstalled() {
			t.Errorf("expected %s to be absent", binary)
		}
		if got := find(); got != binary {
			t.Errorf("expected bare %q, got %q", binary, got)
		}
	})
	t.Run("prefers a runnable custom path", func(t *testing.T) {
		custom := filepath.Join(t.TempDir(), "my-"+binary)
		writeProgram(t, custom)

		if !isInstalled(custom) {
			t.Errorf("expected custom %s to be installed", binary)
		}
		if got := find(custom); got != custom {
			t.Errorf("expected %s, got %s", custom, got)
		}
	})
}

// writeProgram writes an executable shell script that exits at once.
func writeProgram(t *testing.T, path string) {
	t.Helper()
	NeedsShell(t)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}
