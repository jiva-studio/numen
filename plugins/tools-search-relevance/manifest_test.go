package searchrelevance

import (
	"encoding/json"
	"slices"
	"testing"
)

type manifestFile struct {
	ID               string   `json:"id"`
	Capabilities     []string `json:"capabilities"`
	Permissions      []string `json:"permissions"`
	ActivationEvents []string `json:"activationEvents"`
	Targets          []string `json:"targets"`
	Services         []string `json:"services"`
	Backend          struct {
		ServiceName string `json:"serviceName"`
	} `json:"backend"`
	Frontend struct {
		Entrypoint string `json:"entrypoint"`
	} `json:"frontend"`
	SettingsViews []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	} `json:"settingsViews"`
	Settings []struct {
		Key string `json:"key"`
	} `json:"settings"`
}

func readManifest(t *testing.T) manifestFile {
	t.Helper()
	var manifest manifestFile
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}

// The plugin reads the search index and sets the floor, and provides no model of its own.
func TestManifestDeclaresTheIndexAndTheFloorPermissionsAndNoCapability(t *testing.T) {
	manifest := readManifest(t)
	if manifest.ID != "md.numen.tools.search.relevance" {
		t.Errorf("id = %q", manifest.ID)
	}
	if !slices.Equal(manifest.Permissions, []string{"index:read", "search:policy", "vault:read"}) {
		t.Errorf("permissions = %v, want index:read, search:policy and vault:read", manifest.Permissions)
	}
	if len(manifest.Capabilities) != 0 {
		t.Errorf("capabilities = %v, want none: the plugin embeds through the model of the host", manifest.Capabilities)
	}
}

// The events of the host reach the plugin, which measures a vault once its vectors exist.
func TestManifestSubscribesToEveryEvent(t *testing.T) {
	if manifest := readManifest(t); !slices.Equal(manifest.ActivationEvents, []string{"*"}) {
		t.Errorf("activationEvents = %v, want *", manifest.ActivationEvents)
	}
}

// The window reaches the plugin through the service its manifest names.
func TestManifestServesTheRelevanceService(t *testing.T) {
	manifest := readManifest(t)
	if !slices.Equal(manifest.Services, []string{"numen.relevance"}) || manifest.Backend.ServiceName != "" {
		t.Errorf("services = %v and serviceName = %q, want numen.relevance only", manifest.Services, manifest.Backend.ServiceName)
	}
}

// The settings page mounts the relevance panel, which is the whole of the plugin's settings.
func TestManifestDeclaresTheFloorSettingsViewAndNoSetting(t *testing.T) {
	manifest := readManifest(t)
	if !slices.Contains(manifest.Targets, "desktop.editor") || manifest.Frontend.Entrypoint == "" {
		t.Errorf("targets %v and frontend %q, want desktop.editor and an entrypoint", manifest.Targets, manifest.Frontend.Entrypoint)
	}
	if len(manifest.SettingsViews) != 1 || manifest.SettingsViews[0].ID != "floor" || manifest.SettingsViews[0].Title != "Search relevance" {
		t.Fatalf("settingsViews = %+v, want the floor view titled Search relevance", manifest.SettingsViews)
	}
	if len(manifest.Settings) != 0 {
		t.Errorf("settings = %+v, want none: the floor is kept in the host", manifest.Settings)
	}
}
