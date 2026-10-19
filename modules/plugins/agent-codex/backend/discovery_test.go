package main

import (
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
	"os"
	"path/filepath"
	"testing"
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
	if got := FindCodex(); got != "codex" {
		t.Errorf("expected bare 'codex', got %q", got)
	}
}
