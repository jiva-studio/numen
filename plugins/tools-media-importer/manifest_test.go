package downloader

import (
	"encoding/json"
	"slices"
	"strings"
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

// Captions are downloaded for a link note on its own only while the setting for it is on, which is
// off by default.
func TestManifestDownloadsCaptionsAutomaticallyOnlyWithItsSetting(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Triggers []struct {
				On      string `json:"on"`
				Command string `json:"command"`
				When    string `json:"when"`
			} `json:"triggers"`
		} `json:"contributes"`
		Settings []struct {
			Key     string `json:"key"`
			Default any    `json:"default"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	triggers := manifest.Contributes.Triggers
	if len(triggers) != 1 || !strings.Contains(triggers[0].When, "config.md.numen.tools.media.importer.downloadCaptionsAutomatically == true") {
		t.Errorf("the triggers are %+v", triggers)
	}
	found := slices.IndexFunc(manifest.Settings, func(one struct {
		Key     string `json:"key"`
		Default any    `json:"default"`
	}) bool {
		return one.Key == "downloadCaptionsAutomatically"
	})
	if found < 0 || manifest.Settings[found].Default != false {
		t.Errorf("the setting downloadCaptionsAutomatically is missing or on by default: %+v", manifest.Settings)
	}
}
