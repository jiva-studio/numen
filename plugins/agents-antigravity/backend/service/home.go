package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// Endpoint is the address at which the agent calls the tools of this vault over MCP.
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
	if err := agentrun.LinkEntries(realHome, tempHome, ".gemini"); err != nil {
		cleanup()
		return "", func() {}, err
	}

	realGemini := filepath.Join(realHome, ".gemini")
	tempGemini := filepath.Join(tempHome, ".gemini")
	realConfig := filepath.Join(realGemini, "config")
	tempConfig := filepath.Join(tempGemini, "config")
	if err := agentrun.CreateDir(tempConfig, 0o700); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("create %s: %w", tempConfig, err)
	}
	if err := agentrun.LinkEntries(realGemini, tempGemini, "config"); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if err := agentrun.LinkEntries(realConfig, tempConfig, "mcp_config.json"); err != nil {
		cleanup()
		return "", func() {}, err
	}

	mcpConfig := map[string]any{"mcpServers": map[string]any{}}
	if tools != nil {
		mcpConfig = makeMCPConfig(*tools, filepath.Join(realConfig, "mcp_config.json"))
	}
	encoded, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("encode the MCP configuration: %w", err)
	}
	configPath := filepath.Join(tempConfig, "mcp_config.json")
	if err := agentrun.WriteFile(configPath, append(encoded, '\n'), 0o600); err != nil {
		cleanup()
		return "", func() {}, fmt.Errorf("write %s: %w", configPath, err)
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
