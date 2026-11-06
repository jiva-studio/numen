package transcription

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The plugin places transcribing a recording in the menu of a row of the file tree.
func TestManifestPlacesTranscriptionInTheMenuOfARow(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Slots []map[string]any `json:"slots"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{
		{"slot": "fileTree.context", "command": "transcript.create", "group": "3-text", "order": float64(20)},
	}
	if !reflect.DeepEqual(manifest.Contributes.Slots, want) {
		t.Errorf("slots = %v, want %v", manifest.Contributes.Slots, want)
	}
}
