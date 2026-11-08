package transcription

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin declares the capability tools, so the core lists transcript_create to an agent.
func TestManifestDeclaresTheToolsCapability(t *testing.T) {
	var manifest struct {
		Capabilities []string `json:"capabilities"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("the capabilities are %v, want tools among them", manifest.Capabilities)
	}
}
