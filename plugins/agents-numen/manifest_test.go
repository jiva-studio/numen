package agent

import (
	"encoding/json"
	"slices"
	"testing"
)

type settingShape struct {
	Key        string   `json:"key"`
	Type       string   `json:"type"`
	Operations []string `json:"operations"`
	Max        *float64 `json:"max"`
	Default    any      `json:"default"`
}

func readSettings(t *testing.T) []settingShape {
	t.Helper()
	var manifest struct {
		Settings []settingShape `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Settings
}

// The agent chats through one connection, a row that names it, its model and its reasoning level.
func TestManifestDeclaresOneRowOfTheOperationChat(t *testing.T) {
	for _, one := range readSettings(t) {
		if one.Key == "connections" {
			if one.Type != "connections" || !slices.Equal(one.Operations, []string{"llm.chat"}) || one.Max == nil || *one.Max != 1 {
				t.Errorf("the connections setting is %+v, want the kind connections for llm.chat with at most one row", one)
			}
			return
		}
	}
	t.Error("the plugin declares no connections setting")
}

// The connection holds the address, the key and the model, so the plugin keeps none of them.
func TestManifestDeclaresNoAddressKeyOrModelAndNoSecrets(t *testing.T) {
	for _, one := range readSettings(t) {
		if slices.Contains([]string{"baseUrl", "apiKey", "model"}, one.Key) {
			t.Errorf("the plugin declares the setting %s", one.Key)
		}
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if _, hasSecrets := manifest["secrets"]; hasSecrets {
		t.Error("the plugin declares secrets")
	}
}

// The plugin reaches no service of its own: the core does, through the connection.
func TestManifestAsksForNoNetworkPermission(t *testing.T) {
	var manifest struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(manifest.Permissions, "network:outbound") {
		t.Errorf("the manifest declares the permissions %v", manifest.Permissions)
	}
}

// The step limit is a setting of the plugin, 25 requests by default.
func TestManifestDeclaresTheStepLimit(t *testing.T) {
	for _, one := range readSettings(t) {
		if one.Key == "maxSteps" {
			if one.Type != "number" || one.Default != float64(25) {
				t.Errorf("maxSteps is %s with default %v", one.Type, one.Default)
			}
			return
		}
	}
	t.Error("the plugin declares no maxSteps setting")
}
