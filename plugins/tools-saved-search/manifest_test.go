package savedsearch_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	savedsearch "github.com/jiva-studio/numen/plugins/tools-saved-search"
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

	if m.ID != "md.numen.tools.saved.search" || m.Name != "Saved Search" || m.Version != "2026.11.1" || m.Contributes.Category != "tools" {
		t.Errorf("manifest = %+v, want md.numen.tools.saved.search, Saved Search, 2026.11.1 in tools", m)
	}
	if !slices.Equal(m.Capabilities, []string{"tools"}) {
		t.Errorf("capabilities = %v, want [tools]", m.Capabilities)
	}
	if !slices.Equal(m.Permissions, []string{"vault:read", "vault:write", "index:read"}) {
		t.Errorf("permissions = %v, want vault:read, vault:write and index:read", m.Permissions)
	}
	if m.Backend.Entrypoint != "bin/tools-saved-search" {
		t.Errorf("entrypoint = %q, want bin/tools-saved-search", m.Backend.Entrypoint)
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
	if !slices.Equal(m.Services, []string{"md.numen.tools.saved.search"}) {
		t.Errorf("services = %v, want [md.numen.tools.saved.search]", m.Services)
	}
}

func TestManifestDeclaresTheRefreshDelayAndTheDefaultFolderOnly(t *testing.T) {
	m := readManifest(t)

	keys := make([]string, len(m.Settings))
	for i, setting := range m.Settings {
		keys[i] = setting.Key
	}
	if !slices.Equal(keys, []string{"refreshMinutes", "defaultFolder"}) {
		t.Errorf("settings = %v, want refreshMinutes and defaultFolder", keys)
	}
	refresh := findSetting(t, m, "refreshMinutes")
	if refresh.Type != "number" || refresh.Default != float64(10) || refresh.Min == nil || *refresh.Min != 0 {
		t.Errorf("refreshMinutes = %+v, want a number of 10 by default and at least 0", refresh)
	}
	if strings.Contains(refresh.Details, "rerank") || strings.Contains(refresh.Details, "limit") {
		t.Errorf("refreshMinutes details = %q, want no rerank text or limit", refresh.Details)
	}
}

func TestManifestDeclaresTheDefaultFolderAsTheVaultFolder(t *testing.T) {
	folder := findSetting(t, readManifest(t), "defaultFolder")

	if folder.Type != "string" || folder.Default != "" || folder.Title == "" {
		t.Errorf("defaultFolder = %+v, want a titled string setting whose default is the vault folder", folder)
	}
}
