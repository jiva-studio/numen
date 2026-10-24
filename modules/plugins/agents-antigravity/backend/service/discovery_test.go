package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/agent/agentrun/agenttest"
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

	if IsAntigravityInstalled() {
		t.Error("expected Antigravity to be absent")
	}
	if got := FindAntigravity(); got != "agy" {
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
