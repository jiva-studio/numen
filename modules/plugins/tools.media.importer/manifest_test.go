package downloader

import (
	"encoding/json"
	"testing"
)

// The plugin declares yt_dlp and ffmpeg settings replacing core importing settings.
func TestManifestDeclaresDownloaderSettings(t *testing.T) {
	var manifest struct {
		Settings []struct {
			Key      string   `json:"key"`
			Type     string   `json:"type"`
			Replaces []string `json:"replaces"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	found := make(map[string][]string)
	for _, one := range manifest.Settings {
		found[one.Key] = one.Replaces
	}
	if replaces, ok := found["yt_dlp"]; !ok || len(replaces) != 1 || replaces[0] != "importing.yt_dlp" {
		t.Errorf("yt_dlp replaces = %v, want [importing.yt_dlp]", replaces)
	}
	if replaces, ok := found["ffmpeg"]; !ok || len(replaces) != 1 || replaces[0] != "importing.ffmpeg" {
		t.Errorf("ffmpeg replaces = %v, want [importing.ffmpeg]", replaces)
	}
}
