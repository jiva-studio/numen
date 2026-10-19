package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDiscoveryWithMockBins(t *testing.T) {
	binDir := t.TempDir()
	tools := []string{"agy", "codex", "cursor", "claude"}
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
	if !IsClaudeInstalled() {
		t.Error("expected Claude to be installed")
	}

	if got := FindAntigravity(); got != filepath.Join(binDir, "agy") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "agy"), got)
	}
	if got := FindCodex(); got != filepath.Join(binDir, "codex") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "codex"), got)
	}
	if got := FindCursor(); got != filepath.Join(binDir, "cursor") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "cursor"), got)
	}
	if got := FindClaude(); got != filepath.Join(binDir, "claude") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "claude"), got)
	}

	agents := GetInstalledAgents()
	for _, expected := range []string{"antigravity", "codex", "cursor"} {
		if !slices.Contains(agents, expected) {
			t.Errorf("expected installed agents to contain %s, got %v", expected, agents)
		}
	}
}

func TestDiscoveryWithoutAgents(t *testing.T) {
	withoutAgentsAnywhere(t)

	if IsClaudeInstalled() {
		t.Error("expected Claude not to be installed")
	}
	if got := FindCodex(); got != "codex" {
		t.Errorf("expected bare 'codex', got %q", got)
	}
	if got := FindClaude(); got != "claude" {
		t.Errorf("expected bare 'claude', got %q", got)
	}
	if got := FindAntigravity(); got != "agy" {
		t.Errorf("expected bare 'agy', got %q", got)
	}
	if got := FindCursor(); got != "cursor" {
		t.Errorf("expected bare 'cursor', got %q", got)
	}
}
