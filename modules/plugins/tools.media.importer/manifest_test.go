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

// The plugin declares yt-dlp and ffmpeg as system binaries without a download source.
func TestManifestDeclaresBinaryDependencies(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID        string            `json:"id"`
			Kind      string            `json:"kind"`
			SourceURL string            `json:"sourceUrl"`
			Metadata  map[string]string `json:"metadata"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"yt-dlp": "NUMEN_DEPENDENCY_YT_DLP", "ffmpeg": "NUMEN_DEPENDENCY_FFMPEG"}
	if len(manifest.Dependencies) != len(want) {
		t.Fatalf("dependencies = %d, want %d", len(manifest.Dependencies), len(want))
	}
	for _, one := range manifest.Dependencies {
		if one.Kind != "binary" || one.SourceURL != "" || one.Metadata["envVar"] != want[one.ID] || one.Metadata["binary"] != one.ID {
			t.Errorf("dependency %+v does not match the system binary declaration", one)
		}
	}
}
