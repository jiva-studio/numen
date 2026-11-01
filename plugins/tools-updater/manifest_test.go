package updater

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

// The plugin runs in the editor, asks the network for the release feed, and draws one view in the
// status bar.
func TestTheManifestDeclaresTheStatusViewTheTargetAndThePermission(t *testing.T) {
	manifest := plugintest.ReadManifest(t, Manifest)

	if !slices.Equal(manifest.Targets, []string{"desktop.editor"}) {
		t.Errorf("targets = %v", manifest.Targets)
	}
	if !slices.Equal(manifest.Permissions, []string{"network:outbound"}) {
		t.Errorf("permissions = %v", manifest.Permissions)
	}
	if !slices.Equal(manifest.Services, []string{"md.numen.tools.updater"}) {
		t.Errorf("services = %v", manifest.Services)
	}
	if len(manifest.Contributes.StatusViews) != 1 || manifest.Contributes.StatusViews[0].ID != "update" {
		t.Errorf("status views = %+v", manifest.Contributes.StatusViews)
	}
}

// The settings are the ones the backend and the frontend read.
func TestTheManifestDeclaresTheSettingsThePluginReads(t *testing.T) {
	var keys []string
	for _, one := range plugintest.ReadManifest(t, Manifest).Settings {
		keys = append(keys, one.Key)
	}
	slices.Sort(keys)

	if !slices.Equal(keys, []string{"channel", "checkIntervalHours", "endpoint"}) {
		t.Errorf("settings = %v", keys)
	}
}
