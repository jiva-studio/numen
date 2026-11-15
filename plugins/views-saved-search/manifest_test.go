package savedsearch_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	savedsearch "github.com/jiva-studio/numen/plugins/views-saved-search"
)

type manifestSetting struct {
	Key        string   `json:"key"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Operations []string `json:"operations"`
	Default    any      `json:"default"`
	Min        *float64 `json:"min"`
	Max        *float64 `json:"max"`
	Details    string   `json:"details"`
}

type manifestNoteType struct {
	Type   string `json:"type"`
	Chunks *bool  `json:"chunks"`
}

type manifestDoc struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Capabilities     []string          `json:"capabilities"`
	Permissions      []string          `json:"permissions"`
	ActivationEvents []string          `json:"activationEvents"`
	Services         []string          `json:"services"`
	Settings         []manifestSetting `json:"settings"`
	Contributes      struct {
		Category  string             `json:"category"`
		NoteTypes []manifestNoteType `json:"noteTypes"`
	} `json:"contributes"`
	Backend struct {
		Entrypoint string `json:"entrypoint"`
	} `json:"backend"`
}

func readManifest(t *testing.T) manifestDoc {
	t.Helper()
	var m manifestDoc
	if err := json.Unmarshal(savedsearch.Manifest, &m); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	return m
}

func findSetting(t *testing.T, m manifestDoc, key string) manifestSetting {
	t.Helper()
	i := slices.IndexFunc(m.Settings, func(s manifestSetting) bool { return s.Key == key })
	if i < 0 {
		t.Fatalf("settings = %+v, want %s", m.Settings, key)
	}
	return m.Settings[i]
}

func TestManifest(t *testing.T) {
	m := readManifest(t)

	if m.ID != "md.numen.views.saved.search" || m.Name != "Saved Search" || m.Version != "2026.11.3" || m.Contributes.Category != "views" {
		t.Errorf("manifest = %+v, want md.numen.views.saved.search, Saved Search, 2026.11.3 in views", m)
	}
	if !slices.Equal(m.Capabilities, []string{"tools"}) {
		t.Errorf("capabilities = %v, want [tools]", m.Capabilities)
	}
	if !slices.Equal(m.Permissions, []string{"vault:read", "vault:write", "index:read"}) {
		t.Errorf("permissions = %v, want vault:read, vault:write and index:read", m.Permissions)
	}
	if m.Backend.Entrypoint != "bin/views-saved-search" {
		t.Errorf("entrypoint = %q, want bin/views-saved-search", m.Backend.Entrypoint)
	}
}

func TestManifestDeclaresTheSearchNoteTypeWithoutChunks(t *testing.T) {
	m := readManifest(t)

	types := m.Contributes.NoteTypes
	if len(types) != 1 || types[0].Type != "search" || types[0].Chunks == nil || *types[0].Chunks {
		t.Errorf("note types = %+v, want search with chunks false", types)
	}
}

func TestManifestActivatesOnIndexChangesAndDeclaresTheService(t *testing.T) {
	m := readManifest(t)

	if !slices.Equal(m.ActivationEvents, []string{"onIndexChanged"}) {
		t.Errorf("activation events = %v, want [onIndexChanged]", m.ActivationEvents)
	}
	if !slices.Equal(m.Services, []string{"md.numen.views.saved.search"}) {
		t.Errorf("services = %v, want [md.numen.views.saved.search]", m.Services)
	}
}

func TestManifestDeclaresTheRerankingConnections(t *testing.T) {
	connections := findSetting(t, readManifest(t), "connections")

	if connections.Type != "connections" || connections.Title != "Reranking" || !slices.Equal(connections.Operations, []string{"llm.rerank"}) {
		t.Errorf("connections = %+v, want type connections, title Reranking and operation llm.rerank", connections)
	}
}

func TestManifestRecommendsMongoDBRerankThroughOpenRouterAndTheLocalRerankerOtherwise(t *testing.T) {
	details := findSetting(t, readManifest(t), "connections").Details

	for _, part := range []string{"OpenRouter", "mongodb/rerank-3", "Local reranker"} {
		if !strings.Contains(details, part) {
			t.Errorf("connections details = %q, want %q named", details, part)
		}
	}
}

func TestManifestCapsThePassagesPerConditionWithAConnectionAtTheMaximumOfTheIndex(t *testing.T) {
	setting := findSetting(t, readManifest(t), "rerankResultsPerSearch")

	if setting.Max == nil || *setting.Max != 500 {
		t.Errorf("rerankResultsPerSearch = %+v, want a maximum of 500", setting)
	}
}

func TestManifestDeclaresTheNumbersAsSettingsWithTheirDefaults(t *testing.T) {
	want := map[string]float64{
		"defaultLimit": 20, "maxLimit": 100, "resultsPerSearch": 100, "maxCandidates": 1000, "maxPathPrefixes": 8,
		"rerankResultsPerSearch": 300, "fusedPerResult": 3, "maxFused": 100, "rerankedPerResult": 10, "maxReranked": 300,
		"maxRerankedBytes": 4096, "mergeGapBytes": 200, "maxQueryBytes": 16384, "refreshMinutes": 10,
	}
	m := readManifest(t)

	got := map[string]float64{}
	for _, setting := range m.Settings {
		if setting.Type != "number" {
			continue
		}
		number, isNumber := setting.Default.(float64)
		if !isNumber || setting.Min == nil {
			t.Errorf("setting %+v, want a number with a default and a least value", setting)
			continue
		}
		got[setting.Key] = number
	}
	if len(got) != len(want) {
		t.Errorf("number settings = %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("default of %s = %v, want %v", key, got[key], value)
		}
	}
	if refresh := findSetting(t, m, "refreshMinutes"); refresh.Min == nil || *refresh.Min != 0 {
		t.Errorf("refreshMinutes = %+v, want a least value of 0", refresh)
	}
}

func TestManifestDeclaresTheDefaultFolderAsTheVaultFolder(t *testing.T) {
	folder := findSetting(t, readManifest(t), "defaultFolder")

	if folder.Type != "string" || folder.Default != "" || folder.Title == "" {
		t.Errorf("defaultFolder = %+v, want a titled string setting whose default is the vault folder", folder)
	}
}

func TestManifestDeclaresOnlyKnownSettings(t *testing.T) {
	m := readManifest(t)

	if len(m.Settings) != 16 {
		t.Errorf("%d settings, want connections, fourteen numbers and defaultFolder", len(m.Settings))
	}
}
