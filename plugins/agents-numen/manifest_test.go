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

type fieldShape struct {
	Key     string   `json:"key"`
	Default *float64 `json:"default"`
	Min     *float64 `json:"min"`
	Max     *float64 `json:"max"`
}

type connectionTypeShape struct {
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Operations []string          `json:"operations"`
	RowFields  []json.RawMessage `json:"rowFields"`
	Fields     []fieldShape      `json:"fields"`
}

type operationShape struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func readContributes(t *testing.T) (types []connectionTypeShape, operations []operationShape, services []string, capabilities []string) {
	t.Helper()
	var manifest struct {
		Contributes struct {
			ConnectionTypes []connectionTypeShape `json:"connectionTypes"`
			Operations      []operationShape      `json:"operations"`
		} `json:"contributes"`
		Services     []string `json:"services"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Contributes.ConnectionTypes, manifest.Contributes.Operations, manifest.Services, manifest.Capabilities
}

// The agent is a connection of the type numen that runs agent tasks, one at a time unless the person
// allows up to eight.
func TestManifestProvidesTheConnectionTypeNumenForAgentTasks(t *testing.T) {
	types, _, services, capabilities := readContributes(t)

	if len(types) != 1 || types[0].ID != "numen" || types[0].Title != "Numen agent" ||
		!slices.Equal(types[0].Operations, []string{"acp.task"}) || len(types[0].RowFields) != 0 {
		t.Fatalf("the connection types are %+v", types)
	}
	if fields := types[0].Fields; len(fields) != 1 || fields[0].Key != "parallelRequests" ||
		fields[0].Default == nil || fields[0].Min == nil || fields[0].Max == nil || *fields[0].Default != 1 || *fields[0].Min != 1 || *fields[0].Max != 8 {
		t.Errorf("the fields are %+v", fields)
	}
	if !slices.Equal(services, []string{"numen.v1.ProviderService"}) || len(capabilities) != 0 {
		t.Errorf("the services are %v and the capabilities %v", services, capabilities)
	}
}

func TestManifestTitlesTheOperationOfAgentTasks(t *testing.T) {
	_, operations, _, _ := readContributes(t)

	if len(operations) != 1 || operations[0] != (operationShape{Name: "acp.task", Title: "run agent tasks"}) {
		t.Errorf("the operations are %+v", operations)
	}
}
