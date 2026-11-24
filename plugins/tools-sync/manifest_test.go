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
