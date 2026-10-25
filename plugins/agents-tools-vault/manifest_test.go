package vaulttools_test

import (
	"encoding/json"
	"testing"

	vaulttools "github.com/jiva-studio/numen/plugins/agents-tools-vault"
)

type manifestDoc struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Protocol     int      `json:"protocol"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	Targets      []string `json:"targets"`
	Permissions  []string `json:"permissions"`
	Backend      struct {
		Type       string `json:"type"`
		Entrypoint string `json:"entrypoint"`
	} `json:"backend"`
}

func TestManifest(t *testing.T) {
	if len(vaulttools.Manifest) == 0 {
		t.Fatal("manifest is empty")
	}

	var m manifestDoc
	if err := json.Unmarshal(vaulttools.Manifest, &m); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if m.ID != "md.numen.agents.tools.vault" {
		t.Errorf("manifest id = %q, want %q", m.ID, "md.numen.agents.tools.vault")
	}
	if m.Backend.Entrypoint != "bin/agents-tools-vault" {
		t.Errorf("manifest entrypoint = %q, want %q", m.Backend.Entrypoint, "bin/agents-tools-vault")
	}
	if len(m.Capabilities) == 0 || m.Capabilities[0] != "tools" {
		t.Errorf("manifest capabilities = %v, want [tools]", m.Capabilities)
	}
}
