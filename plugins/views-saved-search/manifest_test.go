package savedsearch_test

import (
	"encoding/json"
	"slices"
	"testing"

	savedsearch "github.com/jiva-studio/numen/plugins/views-saved-search"
)

type manifestSetting struct {
	Key        string   `json:"key"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Operations []string `json:"operations"`
	Default    *float64 `json:"default"`
	Min        *float64 `json:"min"`
}

type manifestDoc struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Capabilities []string          `json:"capabilities"`
	Permissions  []string          `json:"permissions"`
	Settings     []manifestSetting `json:"settings"`
	Contributes  struct {
		Category string `json:"category"`
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

func TestManifest(t *testing.T) {
	m := readManifest(t)

	if m.ID != "md.numen.views.saved.search" || m.Name != "Saved Search" || m.Version != "2026.11.1" || m.Contributes.Category != "views" {
		t.Errorf("manifest = %+v, want md.numen.views.saved.search, Saved Search, 2026.11.1 in views", m)
	}
	if !slices.Equal(m.Capabilities, []string{"tools"}) {
		t.Errorf("capabilities = %v, want [tools]", m.Capabilities)
	}
	if !slices.Equal(m.Permissions, []string{"vault:read", "index:read"}) {
		t.Errorf("permissions = %v, want vault:read and index:read", m.Permissions)
	}
	if m.Backend.Entrypoint != "bin/views-saved-search" {
		t.Errorf("entrypoint = %q, want bin/views-saved-search", m.Backend.Entrypoint)
	}
}

func TestManifestDeclaresTheRerankingConnections(t *testing.T) {
	m := readManifest(t)

	i := slices.IndexFunc(m.Settings, func(s manifestSetting) bool { return s.Key == "connections" })
	if i < 0 {
		t.Fatalf("settings = %+v, want connections", m.Settings)
	}
	connections := m.Settings[i]
	if connections.Type != "connections" || connections.Title != "Reranking" || !slices.Equal(connections.Operations, []string{"llm.rerank"}) {
		t.Errorf("connections = %+v, want type connections, title Reranking and operation llm.rerank", connections)
	}
}

func TestManifestDeclaresTheNumbersAsSettingsWithTheirDefaults(t *testing.T) {
	want := map[string]float64{
		"defaultLimit": 20, "maxLimit": 100, "resultsPerSearch": 100, "maxCandidates": 1000, "maxPathPrefixes": 8,
		"rerankedPerResult": 3, "maxReranked": 100, "mergeGapBytes": 200, "maxQueryBytes": 16384,
	}
	m := readManifest(t)

	got := map[string]float64{}
	for _, setting := range m.Settings {
		if setting.Key == "connections" {
			continue
		}
		if setting.Type != "number" || setting.Default == nil || setting.Min == nil {
			t.Errorf("setting %+v, want a number with a default and a least value", setting)
			continue
		}
		got[setting.Key] = *setting.Default
	}
	if len(got) != len(want) {
		t.Errorf("number settings = %v, want %v", got, want)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("default of %s = %v, want %v", key, got[key], value)
		}
	}
}
