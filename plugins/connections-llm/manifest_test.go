package connectionsllm

import (
	"encoding/json"
	"slices"
	"testing"
)

type field struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	OptionsFrom string   `json:"optionsFrom"`
	IsAddress   bool     `json:"isAddress"`
	IsRequired  bool     `json:"required"`
	FixedValue  *string  `json:"fixedValue"`
	Default     *float64 `json:"default"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Step        *float64 `json:"step"`
}

type connectionType struct {
	ID         string   `json:"id"`
	Operations []string `json:"operations"`
	Fields     []field  `json:"fields"`
	RowFields  []field  `json:"rowFields"`
}

type manifestShape struct {
	ID       string   `json:"id"`
	Services []string `json:"services"`
	Backend  struct {
		Entrypoint string `json:"entrypoint"`
	} `json:"backend"`
	Contributes struct {
		Category        string           `json:"category"`
		ConnectionTypes []connectionType `json:"connectionTypes"`
	} `json:"contributes"`
	Settings []struct {
		Key string   `json:"key"`
		Min *float64 `json:"min"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var shape manifestShape
	if err := json.Unmarshal(Manifest, &shape); err != nil {
		t.Fatal(err)
	}
	return shape
}

func getType(t *testing.T, id string) connectionType {
	t.Helper()
	for _, one := range readManifest(t).Contributes.ConnectionTypes {
		if one.ID == id {
			return one
		}
	}
	t.Fatalf("the manifest has no connection type %s", id)
	return connectionType{}
}

func getField(t *testing.T, kind connectionType, key string) field {
	t.Helper()
	for _, one := range kind.Fields {
		if one.Key == key {
			return one
		}
	}
	t.Fatalf("type %s has no field %s", kind.ID, key)
	return field{}
}

func TestManifestProvidesTheThreeTypesOfAnOpenAICompatibleService(t *testing.T) {
	var ids []string
	for _, one := range readManifest(t).Contributes.ConnectionTypes {
		ids = append(ids, one.ID)
		if !slices.Equal(one.Operations, []string{"llm.complete", "llm.chat", "llm.embed", "llm.profile", "llm.models", "llm.reasoning"}) {
			t.Errorf("type %s operations = %v", one.ID, one.Operations)
		}
	}
	if want := []string{"openrouter", "openai", "openai-compatible"}; !slices.Equal(ids, want) {
		t.Errorf("types = %v, want %v", ids, want)
	}
}

func TestManifestDeclaresTheProviderServiceAndABackend(t *testing.T) {
	shape := readManifest(t)
	if shape.ID != "md.numen.connections.llm" || shape.Contributes.Category != "connections" {
		t.Errorf("id = %q, category = %q", shape.ID, shape.Contributes.Category)
	}
	if !slices.Equal(shape.Services, []string{"numen.v1.ProviderService"}) {
		t.Errorf("services = %v", shape.Services)
	}
	if shape.Backend.Entrypoint != "bin/connections-llm" {
		t.Errorf("backend = %q", shape.Backend.Entrypoint)
	}
}

func TestTheAddressOfOpenRouterAndOpenAIIsFixed(t *testing.T) {
	for id, want := range map[string]string{"openrouter": "https://openrouter.ai/api/v1", "openai": "https://api.openai.com/v1"} {
		address := getField(t, getType(t, id), "baseUrl")
		if !address.IsAddress || address.Type != "string" || address.FixedValue == nil || *address.FixedValue != want || address.IsRequired {
			t.Errorf("%s: address field = %+v", id, address)
		}
	}
}

func TestTheAddressOfAnOpenAICompatibleServerIsRequiredAndNotFixed(t *testing.T) {
	address := getField(t, getType(t, "openai-compatible"), "baseUrl")
	if !address.IsAddress || !address.IsRequired || address.FixedValue != nil {
		t.Errorf("address field = %+v", address)
	}
}

func TestTheKeyIsASecretFieldRequiredByTheHostedServicesOnly(t *testing.T) {
	for id, isRequired := range map[string]bool{"openrouter": true, "openai": true, "openai-compatible": false} {
		key := getField(t, getType(t, id), "apiKey")
		if key.Type != "secret" || key.IsRequired != isRequired {
			t.Errorf("%s: key field = %+v", id, key)
		}
	}
}

func TestEveryTypeAllowsFourParallelRequestsAndAtMost32(t *testing.T) {
	for _, one := range readManifest(t).Contributes.ConnectionTypes {
		parallel := getField(t, one, "parallelRequests")
		if parallel.Type != "number" || parallel.Default == nil || *parallel.Default != 4 || *parallel.Min != 1 || *parallel.Max != 32 || *parallel.Step != 1 {
			t.Errorf("%s: parallelRequests = %+v", one.ID, parallel)
		}
		if last := one.Fields[len(one.Fields)-1]; last.Key != "parallelRequests" {
			t.Errorf("%s: last field = %s, want parallelRequests", one.ID, last.Key)
		}
	}
}

// A row names the model, listed by llm.models, and the reasoning level, listed by llm.reasoning.
func TestEveryTypeListsTheModelAndTheReasoningOfARowFromItsProvider(t *testing.T) {
	for _, one := range readManifest(t).Contributes.ConnectionTypes {
		if len(one.RowFields) != 2 {
			t.Fatalf("%s: row fields = %+v", one.ID, one.RowFields)
		}
		model, reasoning := one.RowFields[0], one.RowFields[1]
		if model.Key != "model" || !model.IsRequired || model.OptionsFrom != "llm.models" {
			t.Errorf("%s: model row field = %+v", one.ID, model)
		}
		if reasoning.Key != "reasoning" || reasoning.IsRequired || reasoning.OptionsFrom != "llm.reasoning" {
			t.Errorf("%s: reasoning row field = %+v", one.ID, reasoning)
		}
	}
}

func TestManifestDeclaresTheSettingRequestTimeout(t *testing.T) {
	var found bool
	for _, one := range readManifest(t).Settings {
		found = found || one.Key == "requestTimeout"
	}
	if !found {
		t.Error("the manifest has no setting requestTimeout")
	}
}

// The provider asks for no permission: it reaches the services of its connections itself, and a
// plugin is not stopped from the network by a permission.
func TestManifestAsksForNoPermission(t *testing.T) {
	var manifest struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Permissions) != 0 {
		t.Errorf("permissions = %v, want none", manifest.Permissions)
	}
}
