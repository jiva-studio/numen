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

func TestManifestDeclaresTheMCPListenerOnThePortSetting(t *testing.T) {
	t.Parallel()

	type listener struct {
		Name        string `json:"name"`
		PortSetting string `json:"portSetting"`
		Protocol    string `json:"protocol"`
	}
	type setting struct {
		Key     string          `json:"key"`
		Type    string          `json:"type"`
		Default json.RawMessage `json:"default"`
	}
	var manifest struct {
		ActivationEvents []string `json:"activationEvents"`
		Contributes      struct {
			Listeners []listener `json:"listeners"`
		} `json:"contributes"`
		Settings []setting `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	if slices.Contains(manifest.ActivationEvents, "*") {
		t.Errorf("activation events %v start the plugin with the application", manifest.ActivationEvents)
	}
	want := []listener{{Name: "mcp", PortSetting: "port", Protocol: "tcp"}}
	if !slices.Equal(manifest.Contributes.Listeners, want) {
		t.Errorf("listeners = %+v, want %+v", manifest.Contributes.Listeners, want)
	}
	index := slices.IndexFunc(manifest.Settings, func(s setting) bool { return s.Key == "port" })
	if index < 0 {
		t.Fatal("manifest declares no setting port")
	}
	if port := manifest.Settings[index]; port.Type != "number" || string(port.Default) != "3333" {
		t.Errorf("setting port = %+v, want a number with default 3333", port)
	}
}

func TestManifestDeclaresTheTokenSettingAsAStringWithoutADefault(t *testing.T) {
	t.Parallel()

	type setting struct {
		Key     string          `json:"key"`
		Type    string          `json:"type"`
		Title   string          `json:"title"`
		Default json.RawMessage `json:"default"`
	}
	var manifest struct {
		Settings []setting `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	index := slices.IndexFunc(manifest.Settings, func(s setting) bool { return s.Key == "token" })
	if index < 0 {
		t.Fatal("manifest declares no setting token")
	}
	token := manifest.Settings[index]
	if token.Type != "string" || token.Title != "Token" {
		t.Errorf("setting token = %+v, want a string titled Token", token)
	}
	if token.Default != nil {
		t.Errorf("setting token has the default %s, want none", token.Default)
	}
}
