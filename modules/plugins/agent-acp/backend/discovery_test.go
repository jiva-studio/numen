package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	binDir := t.TempDir()
	tools := []string{"agy", "codex", "cursor"}
	for _, tool := range tools {
		path := filepath.Join(binDir, tool)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("PATH", binDir)

	if !IsAntigravityInstalled() {
		t.Error("expected Antigravity to be installed")
	}
	if !IsCodexInstalled() {
		t.Error("expected Codex to be installed")
	}
	if !IsCursorInstalled() {
		t.Error("expected Cursor to be installed")
	}

	agents := GetInstalledAgents()
	for _, expected := range []string{"antigravity", "codex", "cursor"} {
		if !slices.Contains(agents, expected) {
			t.Errorf("expected installed agents to contain %s, got %v", expected, agents)
		}
	}
}
