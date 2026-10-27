package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// Endpoint is where the agent reaches the tools of this vault over MCP.
type Endpoint struct {
	URL   string
	Token string
}

// setupAntigravityHome creates a temporary HOME folder with symlinked configs. Its MCP servers are the
// vault's tools and those the user configured, or none at all where there are no tools.
func setupAntigravityHome(tools *Endpoint) (string, func(), error) {
	tempHome, cleanup, err := agentrun.CreateTempDir("numen-antigravity-*")
	if err != nil {
		return "", func() {}, err
	}

	realHome, err := os.UserHomeDir()
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	agentrun.LinkEntries(realHome, tempHome, ".gemini")

	realGemini := filepath.Join(realHome, ".gemini")
	tempGemini := filepath.Join(tempHome, ".gemini")
	_ = agentrun.CreateDir(filepath.Join(tempGemini, "config"), 0o700)
	agentrun.LinkEntries(realGemini, tempGemini, "config")

	mcpConfig := map[string]any{"mcpServers": map[string]any{}}
	if tools != nil {
		mcpConfig = makeMCPConfig(*tools, filepath.Join(realGemini, "config", "mcp_config.json"))
	}
	if encoded, err := json.MarshalIndent(mcpConfig, "", "  "); err == nil {
		_ = agentrun.WriteFile(filepath.Join(tempGemini, "config", "mcp_config.json"), append(encoded, '\n'), 0o600)
	}

	return tempHome, cleanup, nil
}

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
	if raw, err := agentrun.ReadFile(existingPath); err == nil {
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
