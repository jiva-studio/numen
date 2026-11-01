package window

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestManifestIsValid(t *testing.T) {
	var manifest struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Protocol     int      `json:"protocol"`
		Capabilities []string `json:"capabilities"`
		Targets      []string `json:"targets"`
		Permissions  []string `json:"permissions"`
		Backend      struct {
			Type       string `json:"type"`
			Entrypoint string `json:"entrypoint"`
		} `json:"backend"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	if manifest.ID != "md.numen.agents.tools.window" {
		t.Errorf("manifest.ID = %q, want md.numen.agents.tools.window", manifest.ID)
	}
	if !slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("capabilities %v does not contain tools", manifest.Capabilities)
	}
	if !slices.Contains(manifest.Targets, "desktop.editor") {
		t.Errorf("targets %v does not contain desktop.editor", manifest.Targets)
	}
	if !slices.Contains(manifest.Permissions, "vault:read") {
		t.Errorf("permissions %v does not contain vault:read", manifest.Permissions)
	}
	if manifest.Backend.Entrypoint != "bin/agents-tools-window" {
		t.Errorf("backend.entrypoint = %q, want bin/agents-tools-window", manifest.Backend.Entrypoint)
	}
}
