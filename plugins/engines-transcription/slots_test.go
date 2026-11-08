package transcription

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The plugin places its handler of creating text in the menu of a row of the file tree.
func TestManifestPlacesItsHandlerInTheMenuOfARow(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Slots []map[string]any `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{
		{"slot": "fileTree.context", "command": "app:file.createText", "group": "3-text", "order": float64(10)},
	}
	if !reflect.DeepEqual(manifest.Contributes.Slots, want) {
		t.Errorf("slots = %v, want %v", manifest.Contributes.Slots, want)
	}
}
