package settings_test

import (
	"path/filepath"
	"testing"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
)

func TestLocateFilePrefersTheSettingsPathThenTheRegistryFolder(t *testing.T) {
	registry := filepath.Join("state", "vaults.json")

	if got, err := settingsadapter.LocateFile("custom.json", registry); err != nil || got != "custom.json" {
		t.Errorf("an explicit path: %q, %v", got, err)
	}
	if got, err := settingsadapter.LocateFile("", registry); err != nil || got != filepath.Join("state", "numen.json") {
		t.Errorf("beside the registry: %q, %v", got, err)
	}
}

func TestLocateThemesPrefersTheThemesPathThenTheRegistryFolder(t *testing.T) {
	registry := filepath.Join("state", "vaults.json")

	if got, isSet := settingsadapter.LocateThemes("mine", registry); !isSet || got != "mine" {
		t.Errorf("an explicit path: %q, %v", got, isSet)
	}
	if got, isSet := settingsadapter.LocateThemes("", registry); !isSet || got != filepath.Join("state", "themes") {
		t.Errorf("beside the registry: %q, %v", got, isSet)
	}
	if got, isSet := settingsadapter.LocateThemes("", ""); isSet {
		t.Errorf("no place set returned %q", got)
	}
}
