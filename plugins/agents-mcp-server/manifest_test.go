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
		Services     []string `json:"services"`
	}

	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if manifest.ID != "md.numen.agents.mcp.server" {
		t.Errorf("got ID %q, want %q", manifest.ID, "md.numen.agents.mcp.server")
	}

	// The plugin serves no PluginToolService: the core requests the tools of every plugin with the tools capability
	// when an agent run starts.
	if slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("manifest declares the tools capability: %v", manifest.Capabilities)
	}

	if !slices.Equal(manifest.Services, []string{"md.numen.agents.mcp.server"}) {
		t.Errorf("got services %v, want %q", manifest.Services, "md.numen.agents.mcp.server")
	}
}

func TestManifestDeclaresTheMCPListenerAndNoPortSetting(t *testing.T) {
	t.Parallel()

	type listener struct {
		Name     string `json:"name"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
	}
	var manifest struct {
		ActivationEvents []string `json:"activationEvents"`
		Contributes      struct {
			Listeners []listener `json:"listeners"`
		} `json:"contributes"`
		Settings []struct {
			Key string `json:"key"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if slices.Contains(manifest.ActivationEvents, "*") {
		t.Errorf("activation events %v start the plugin with the application", manifest.ActivationEvents)
	}
	for _, setting := range manifest.Settings {
		if setting.Key == "port" {
			t.Error("manifest declares the setting port")
		}
	}
	want := []listener{{Name: "mcp", Port: 3333, Protocol: "tcp"}}
	if !slices.Equal(manifest.Contributes.Listeners, want) {
		t.Errorf("listeners = %+v, want %+v", manifest.Contributes.Listeners, want)
	}
}
