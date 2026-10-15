package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
)

func TestAgentPortWithoutHost(t *testing.T) {
	cfg := Config{}
	if agent := cfg.AgentPort(); agent != nil {
		t.Fatalf("expected nil agent, got %v", agent)
	}
}

func TestAgentPortWithoutPlugin(t *testing.T) {
	tmp := t.TempDir()
	host := plugin.NewHost("desktop.editor", tmp)
	_ = host.Discover()

	cfg := Config{PluginHost: host}
	if agent := cfg.AgentPort(); agent != nil {
		t.Fatalf("expected nil agent, got %v", agent)
	}
}

func TestAgentPortWithPlugin(t *testing.T) {
	tmp := t.TempDir()
	pDir := filepath.Join(tmp, "agent")
	if err := os.MkdirAll(pDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	manifest := `{
		"id": "md.numen.agent",
		"name": "Agent Plugin",
		"version": "1.0.0",
		"capabilities": ["agent"],
		"targets": ["desktop.editor"],
		"backend": {
			"type": "process",
			"entrypoint": "bin/agent",
			"serviceName": "numen.v1.AgentService"
		}
	}`
	if err := os.WriteFile(filepath.Join(pDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("discover failed: %v", err)
	}

	cfg := Config{PluginHost: host}
	agent := cfg.AgentPort()
	if agent == nil {
		t.Fatalf("expected non-nil agent port")
	}
}
