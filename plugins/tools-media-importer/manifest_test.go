package downloader

import (
	"encoding/json"
	"testing"
)

// The plugin declares no program as a dependency: an article is read without yt-dlp and ffmpeg, so
// a machine without them starts the plugin, and only copies no video.
func TestManifestDeclaresNoProgramTheHostMustFind(t *testing.T) {
	var manifest struct {
		Dependencies []json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Dependencies) != 0 {
		t.Errorf("the plugin declares %d dependencies", len(manifest.Dependencies))
	}
}

// The command that imports an address takes the address and nothing else: the palette asks only
// required arguments, and a caller that is not a person has no option to ask for.
func TestManifestDeclaresOneRequiredArgumentForLinkImport(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Commands []struct {
				ID        string `json:"id"`
				Arguments []struct {
					Name     string `json:"name"`
					Required bool   `json:"required"`
				} `json:"arguments"`
			} `json:"commands"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, one := range manifest.Contributes.Commands {
		if one.ID != "link.import" {
			continue
		}
		if len(one.Arguments) != 1 || one.Arguments[0].Name != "address" || !one.Arguments[0].Required {
			t.Errorf("link.import takes %+v, want the required argument address alone", one.Arguments)
		}
		return
	}
	t.Error("the manifest declares no link.import")
}
