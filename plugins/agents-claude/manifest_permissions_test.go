package agentclaude

import (
	"encoding/json"
	"slices"
	"testing"
)

// The plugin serves the tools of the others to an agent, so it declares the extra
// permissions those tools require.
func TestManifestListsVaultManageWindowFocusAndIndexRead(t *testing.T) {
	var manifest struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"vault:read", "vault:write", "vault:manage", "window:focus", "index:read"} {
		if !slices.Contains(manifest.Permissions, want) {
			t.Errorf("the manifest declares the permissions %v, want %s among them", manifest.Permissions, want)
		}
	}
}
