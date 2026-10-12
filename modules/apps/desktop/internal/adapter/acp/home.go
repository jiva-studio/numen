package acp

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// setupAntigravityHome makes a home folder for the agent that links to the real
// one, and carries the vault's tools in its MCP configuration. The cleanup
// removes it.
func setupAntigravityHome(tools Endpoint) (string, func(), error) {
	tempHome, err := os.MkdirTemp("", "numen-antigravity-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(tempHome) }

	realHome, err := os.UserHomeDir()
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	linkEntries(realHome, tempHome, ".gemini")

	realGemini := filepath.Join(realHome, ".gemini")
	tempGemini := filepath.Join(tempHome, ".gemini")
	_ = os.MkdirAll(filepath.Join(tempGemini, "config"), 0o700)
	linkEntries(realGemini, tempGemini, "config")

	mcpConfig := makeMCPConfig(tools, filepath.Join(realGemini, "config", "mcp_config.json"))
	if encoded, err := json.MarshalIndent(mcpConfig, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(tempGemini, "config", "mcp_config.json"), append(encoded, '\n'), 0o600)
	}

	return tempHome, cleanup, nil
}

// linkEntries links every entry of a folder into another, but the one named
// except.
func linkEntries(from, into, except string) {
	entries, err := os.ReadDir(from)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if name != except {
			_ = os.Symlink(filepath.Join(from, name), filepath.Join(into, name))
		}
	}
}

// makeMCPConfig is the MCP configuration naming the vault's tools, beside the
// servers the existing configuration file names.
func makeMCPConfig(tools Endpoint, existingPath string) map[string]any {
	servers := map[string]any{
		"numen": map[string]any{
			"type":      "http",
			"serverUrl": tools.URL,
			"headers": map[string]string{
				"Authorization": "Bearer " + tools.Token,
			},
		},
	}
	if raw, err := os.ReadFile(existingPath); err == nil {
		var existing struct {
			MCPServers map[string]any `json:"mcpServers"`
		}
		if err := json.Unmarshal(raw, &existing); err == nil {
			for k, v := range existing.MCPServers {
				if k != "numen" {
					servers[k] = v
				}
			}
		}
	}
	return map[string]any{"mcpServers": servers}
}
