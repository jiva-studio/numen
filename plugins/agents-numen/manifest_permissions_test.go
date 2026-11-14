package agent

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// The plugin serves the tools of the others to an agent, so it declares the extra
// permissions those tools require, and it chats through the connections that serve llm.chat and no other operation.
func TestManifestListsVaultManageWindowFocusIndexReadAndConnectionsUseOfLLMChat(t *testing.T) {
	var manifest struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(Manifest, &manifest); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"vault:read", "vault:write", "vault:manage", "window:focus", "index:read", "connections:use:llm.chat"} {
		if !slices.Contains(manifest.Permissions, want) {
			t.Errorf("the manifest declares the permissions %v, want %s among them", manifest.Permissions, want)
		}
	}
	for _, permission := range manifest.Permissions {
		if strings.HasPrefix(permission, "connections:use:") && permission != "connections:use:llm.chat" {
			t.Errorf("the manifest grants %s, want connections only for llm.chat", permission)
		}
	}
}
