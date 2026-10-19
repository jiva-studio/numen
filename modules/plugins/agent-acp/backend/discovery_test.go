package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
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

	if got := FindAntigravity(); got != filepath.Join(binDir, "agy") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "agy"), got)
	}
	if got := FindCodex(); got != filepath.Join(binDir, "codex") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "codex"), got)
	}
	if got := FindCursor(); got != filepath.Join(binDir, "cursor") {
		t.Errorf("expected %s, got %s", filepath.Join(binDir, "cursor"), got)
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

	if got := FindCodex(); got != "codex" {
		t.Errorf("expected bare 'codex', got %q", got)
	}
	if got := FindAntigravity(); got != "agy" {
		t.Errorf("expected bare 'agy', got %q", got)
	}
	if got := FindCursor(); got != "cursor" {
		t.Errorf("expected bare 'cursor', got %q", got)
	}
	if got := GetACPAgents(); len(got) != 0 {
		t.Errorf("expected no ACP agents, got %v", got)
	}
}

// Codex answers over ACP through codex-acp, and Cursor through the acp command of cursor-agent; the
// programs that run tasks answer none.
func TestTheACPAgentsAreTheirACPServers(t *testing.T) {
	needsShell(t)
	withoutAgentsAnywhere(t)
	binDir := t.TempDir()
	for _, tool := range []string{"agy", "codex", "cursor", "codex-acp", "cursor-agent"} {
		if err := os.WriteFile(filepath.Join(binDir, tool), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir)

	var got []string
	for _, agent := range GetACPAgents() {
		got = append(got, agent.Program+"="+filepath.Base(agent.Command[0])+" "+strings.Join(agent.Command[1:], " "))
	}
	want := []string{"codex=codex-acp ", "cursor=cursor-agent acp"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
