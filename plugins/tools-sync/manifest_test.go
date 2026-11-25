package sync_test

import (
	"encoding/json"
	"slices"
	"testing"

	toolssync "github.com/jiva-studio/numen/plugins/tools-sync"
)

func TestManifestTargetsTheEditorAndThePhone(t *testing.T) {
	var manifest struct {
		Targets []string `json:"targets"`
	}
	if err := json.Unmarshal(toolssync.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"desktop.editor", "mobile.app"} {
		if !slices.Contains(manifest.Targets, want) {
			t.Errorf("targets = %v, want %s among them", manifest.Targets, want)
		}
	}
}

func TestManifestDeclaresTheFilesItProvides(t *testing.T) {
	var manifest struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(toolssync.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "files.provider") {
		t.Errorf("capabilities = %v, want files.provider among them", manifest.Capabilities)
	}
}

func TestManifestDownloadsBigFilesOnDemandByDefault(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key     string `json:"key"`
			Type    string `json:"type"`
			Title   string `json:"title"`
			Default any    `json:"default"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(toolssync.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, setting := range manifest.Settings {
		if setting.Key != "downloadOnDemand" {
			continue
		}
		if setting.Type != "boolean" || setting.Default != true || setting.Title != "Download big files when opened" {
			t.Errorf("setting = %+v, want a boolean titled for big files, on by default", setting)
		}
		return
	}
	t.Error("settings have no downloadOnDemand")
}
