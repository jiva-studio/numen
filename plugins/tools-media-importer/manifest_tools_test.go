package downloader

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin declares the capability tools, so the core lists url_import to an agent.
func TestManifestDeclaresTheToolsCapability(t *testing.T) {
	var manifest struct {
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("the capabilities are %v, want tools among them", manifest.Capabilities)
	}
	if manifest.Version != "2026.11.9" {
		t.Errorf("the version is %s, want 2026.11.9", manifest.Version)
	}
}
