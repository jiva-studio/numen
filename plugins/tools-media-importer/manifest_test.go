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

// The plugin declares the commands of links as its own and creates the text of a link as the handler
// of the shared command.
func TestManifestDeclaresTheCommandsOfLinks(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Commands []struct {
				ID string `json:"id"`
			} `json:"commands"`
			Handlers []struct {
				Command string `json:"command"`
			} `json:"handlers"`
		} `json:"contributes"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	var commands, handlers []string
	for _, one := range manifest.Contributes.Commands {
		commands = append(commands, one.ID)
	}
	for _, one := range manifest.Contributes.Handlers {
		handlers = append(handlers, one.Command)
	}
	wantCommands := []string{"link.import", "link.downloadMedia", "link.deleteMedia"}
	if !slices.Equal(commands, wantCommands) || !slices.Equal(handlers, []string{"app:file.createText"}) {
		t.Errorf("commands %v and handlers %v, want %v and app:file.createText", commands, handlers, wantCommands)
	}
}

// Captions are downloaded for a link note on its own only while the setting for it is on, which is
// off by default.
func TestManifestDownloadsCaptionsAutomaticallyOnlyWithItsSetting(t *testing.T) {
	var manifest struct {
		Contributes struct {
			Triggers []struct {
				Event   string `json:"event"`
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
	if len(triggers) != 1 || triggers[0].Event != "file.added" || !strings.Contains(triggers[0].When, "config.md.numen.tools.media.importer.downloadCaptionsAutomatically == true") ||
		!strings.Contains(triggers[0].When, "!target.versions.exists(v, v.pluginId == 'md.numen.tools.media.importer' && v.isValid)") {
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
