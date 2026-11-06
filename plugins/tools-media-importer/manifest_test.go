package downloader

import (
	"encoding/json"
	"slices"
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

// The plugin handles the shared commands of links, and creates the text of a link, as handlers.
func TestManifestHandlesTheSharedCommandsOfLinks(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Commands []json.RawMessage `json:"commands"`
			Handlers []struct {
				Command string `json:"command"`
			} `json:"handlers"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, one := range manifest.Contributes.Handlers {
		got = append(got, one.Command)
	}
	want := []string{"app:link.import", "app:file.createText", "app:link.downloadMedia", "app:link.deleteMedia", "app:version.delete"}
	if !slices.Equal(got, want) || len(manifest.Contributes.Commands) != 0 {
		t.Errorf("handlers %v and %d commands, want %v and none", got, len(manifest.Contributes.Commands), want)
	}
}
