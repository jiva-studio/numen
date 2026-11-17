package reranking

import (
	"encoding/json"
	"slices"
	"testing"
)

type manifestField struct {
	Key         string   `json:"key"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Required    bool     `json:"required"`
	OptionsFrom string   `json:"optionsFrom"`
	Default     any      `json:"default"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Step        *float64 `json:"step"`
}

type manifestConnectionType struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Operations  []string        `json:"operations"`
	RowFields   []manifestField `json:"rowFields"`
	Fields      []manifestField `json:"fields"`
}

type manifestOperation struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type manifestShape struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Protocol    int      `json:"protocol"`
	Description string   `json:"description"`
	Targets     []string `json:"targets"`
	Contributes struct {
		Category        string                   `json:"category"`
		Operations      []manifestOperation      `json:"operations"`
		ConnectionTypes []manifestConnectionType `json:"connectionTypes"`
	} `json:"contributes"`
	Permissions  []string `json:"permissions"`
	Dependencies []struct {
		ID   string `json:"id"`
		Kind string `json:"kind"`
	} `json:"dependencies"`
	Backend struct {
		Entrypoint      string `json:"entrypoint"`
		IdleStopSeconds int    `json:"idleStopSeconds"`
	} `json:"backend"`
	Services         []string          `json:"services"`
	Capabilities     []string          `json:"capabilities"`
	Settings         []json.RawMessage `json:"settings"`
	Frontend         map[string]any    `json:"frontend"`
	ActivationEvents []string          `json:"activationEvents"`
}

func readManifest(t *testing.T) manifestShape {
	t.Helper()
	var manifest manifestShape
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

func TestManifestNamesThePluginLocalReranking(t *testing.T) {
	manifest := readManifest(t)

	if manifest.ID != "md.numen.engines.reranking" || manifest.Name != "Local Reranking" || manifest.Version != "2026.11.4" ||
		manifest.Protocol != 1 || manifest.Description == "" {
		t.Errorf("manifest = %q %q %q protocol %d, description %q", manifest.ID, manifest.Name, manifest.Version, manifest.Protocol, manifest.Description)
	}
	if !slices.Equal(manifest.Targets, []string{"desktop.editor"}) || manifest.Contributes.Category != "engines" {
		t.Errorf("targets %v, category %q, want desktop.editor and engines", manifest.Targets, manifest.Contributes.Category)
	}
}

func TestManifestProvidesTheLocalRerankerType(t *testing.T) {
	manifest := readManifest(t)

	if len(manifest.Contributes.ConnectionTypes) != 1 {
		t.Fatalf("connection types = %+v, want one", manifest.Contributes.ConnectionTypes)
	}
	reranker := manifest.Contributes.ConnectionTypes[0]
	if reranker.ID != "local-reranker" || reranker.Title != "Local reranker" || reranker.Description == "" {
		t.Errorf("type = %q %q %q, want local-reranker titled Local reranker", reranker.ID, reranker.Title, reranker.Description)
	}
	if !slices.Equal(reranker.Operations, []string{"llm.rerank", "llm.models", "connections.detected"}) {
		t.Errorf("operations = %v, want llm.rerank, llm.models and connections.detected", reranker.Operations)
	}
	want := []manifestField{{Key: "model", Type: "string", Title: "Model", OptionsFrom: "llm.models"}}
	if !slices.EqualFunc(reranker.RowFields, want, isSameField) {
		t.Errorf("row fields = %+v, want one optional string model with options from llm.models", reranker.RowFields)
	}
}

func isSameField(a, b manifestField) bool {
	return a.Key == b.Key && a.Type == b.Type && a.Title == b.Title && a.Required == b.Required && a.OptionsFrom == b.OptionsFrom
}

func TestManifestLetsAConnectionRunOneToFourCallsAtOnce(t *testing.T) {
	manifest := readManifest(t)
	if len(manifest.Contributes.ConnectionTypes) != 1 {
		t.Fatalf("connection types = %+v, want one", manifest.Contributes.ConnectionTypes)
	}

	fields := manifest.Contributes.ConnectionTypes[0].Fields

	if len(fields) != 1 {
		t.Fatalf("fields = %+v, want parallelRequests alone", fields)
	}
	one := fields[0]
	if one.Key != "parallelRequests" || one.Type != "number" || one.Title != "Parallel requests" || one.Default != float64(1) ||
		one.Min == nil || *one.Min != 1 || one.Max == nil || *one.Max != 4 || one.Step == nil || *one.Step != 1 {
		t.Errorf("field = %+v, want the number Parallel requests from 1 to 4 by 1 that starts at 1", one)
	}
}

func TestManifestTitlesTheOperationsItServes(t *testing.T) {
	manifest := readManifest(t)

	want := []manifestOperation{{Name: "llm.rerank", Title: "rerank texts"}, {Name: "llm.models", Title: "list models"}}
	if !slices.Equal(manifest.Contributes.Operations, want) {
		t.Errorf("operations = %+v, want %+v", manifest.Contributes.Operations, want)
	}
}

func TestManifestDeclaresONNXRuntimeDependency(t *testing.T) {
	manifest := readManifest(t)

	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0].ID != "onnxruntime" || manifest.Dependencies[0].Kind != "runtime" {
		t.Errorf("dependencies = %+v, want one onnxruntime runtime", manifest.Dependencies)
	}
}

// link:open is the consent docs/plugins/README.md asks of a plugin that downloads a model.
func TestManifestAsksOnlyForTheConsentToDownloadAModel(t *testing.T) {
	manifest := readManifest(t)

	if !slices.Equal(manifest.Permissions, []string{"link:open"}) {
		t.Errorf("permissions = %v, want link:open alone", manifest.Permissions)
	}
}

func TestManifestServesTheProviderServiceFromABackendOnly(t *testing.T) {
	manifest := readManifest(t)

	if manifest.Backend.Entrypoint != "bin/engines-reranking" || !slices.Equal(manifest.Services, []string{"numen.v1.ProviderService"}) {
		t.Errorf("backend %q, services %v, want bin/engines-reranking serving numen.v1.ProviderService", manifest.Backend.Entrypoint, manifest.Services)
	}
	if len(manifest.Capabilities) != 0 || len(manifest.Settings) != 0 || len(manifest.Frontend) != 0 || len(manifest.ActivationEvents) != 0 {
		t.Errorf("capabilities %v, %d settings, frontend %v, activationEvents %v, want none of them",
			manifest.Capabilities, len(manifest.Settings), manifest.Frontend, manifest.ActivationEvents)
	}
}

func TestManifestKeepsTheProcessForTenIdleMinutes(t *testing.T) {
	manifest := readManifest(t)

	if manifest.Backend.IdleStopSeconds != 600 {
		t.Errorf("idleStopSeconds = %d, want 600", manifest.Backend.IdleStopSeconds)
	}
}
