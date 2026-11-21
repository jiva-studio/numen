package proofreading

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The plugin places its command in the menu of a row of the file tree, where it proofreads the
// live text version of the file.
func TestManifestPlacesItsCommandInTheMenuOfARow(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Slots []map[string]any `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{
		{"slot": "fileTree.context", "command": "proofreading.create", "group": "3-text", "order": float64(20)},
	}
	if !reflect.DeepEqual(manifest.Contributes.Slots, want) {
		t.Errorf("slots = %v, want %v", manifest.Contributes.Slots, want)
	}
}
