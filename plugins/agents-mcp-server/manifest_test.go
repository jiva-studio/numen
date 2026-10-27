package agentmcpserver

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestManifestValid(t *testing.T) {
	t.Parallel()

	var manifest struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Protocol     int      `json:"protocol"`
		Capabilities []string `json:"capabilities"`
		Backend      struct {
			Type        string `json:"type"`
			Entrypoint  string `json:"entrypoint"`
			ServiceName string `json:"serviceName"`
		} `json:"backend"`
	}

	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if manifest.ID != "md.numen.agents.mcp.server" {
		t.Errorf("got ID %q, want %q", manifest.ID, "md.numen.agents.mcp.server")
	}

	// The plugin serves no PluginToolService: the core asks every plugin with the tools capability
	// for its tools when an agent run starts.
	if slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("manifest declares the tools capability: %v", manifest.Capabilities)
	}

	if manifest.Backend.ServiceName != "numen.v1.McpServerService" {
		t.Errorf("got ServiceName %q, want %q", manifest.Backend.ServiceName, "numen.v1.McpServerService")
	}
}
