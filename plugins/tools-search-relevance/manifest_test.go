package searchrelevance

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin reads the search index and sets the floor, and provides no model of its own.
func TestManifestDeclaresTheIndexAndTheFloorPermissionsAndNoCapability(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
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
	if manifest := plugintest.ReadManifest(t, Manifest); !slices.Equal(manifest.ActivationEvents, []string{"*"}) {
		t.Errorf("activationEvents = %v, want *", manifest.ActivationEvents)
	}
}

// The window reaches the plugin through the service its manifest names.
func TestManifestServesTheRelevanceService(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
	if !slices.Equal(manifest.Services, []string{"md.numen.tools.search.relevance"}) {
		t.Errorf("services = %v, want md.numen.tools.search.relevance only", manifest.Services)
	}
}

// The settings page mounts the relevance panel, which is the whole of the plugin's settings.
func TestManifestDeclaresTheFloorSettingsViewAndNoSetting(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)
	if !slices.Contains(manifest.Targets, "desktop.editor") || manifest.Frontend == nil {
		t.Errorf("targets %v and frontend %v, want desktop.editor and a frontend", manifest.Targets, manifest.Frontend)
	}
	if len(manifest.SettingsViews) != 1 || manifest.SettingsViews[0].ID != "floor" || manifest.SettingsViews[0].Title != "Search relevance" {
		t.Fatalf("settingsViews = %+v, want the floor view titled Search relevance", manifest.SettingsViews)
	}
	if len(manifest.Settings) != 0 {
		t.Errorf("settings = %+v, want none: the floor is kept in the host", manifest.Settings)
	}
}
