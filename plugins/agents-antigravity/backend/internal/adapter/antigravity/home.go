package antigravity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/agent/agentrun"
)

// endpoint is the address at which the agent calls the tools of this vault over MCP.
type endpoint struct {
	URL   string
	Token string
}

func noRelease() error { return nil }

func setupAntigravityHome(tools *endpoint) (string, func() error, error) {
	tempHome, cleanup, err := agentrun.CreateTempDir("numen-antigravity-*")
	if err != nil {
		return "", noRelease, err
	}

	realHome, err := os.UserHomeDir()
	if err != nil {
		return "", noRelease, errors.Join(err, cleanup())
	}
	if err := agentrun.LinkEntries(realHome, tempHome, ".gemini"); err != nil {
		return "", noRelease, errors.Join(err, cleanup())
	}

	realGemini := filepath.Join(realHome, ".gemini")
	tempGemini := filepath.Join(tempHome, ".gemini")
	realConfig := filepath.Join(realGemini, "config")
	tempConfig := filepath.Join(tempGemini, "config")
	// gomutants:disable-next-line BRANCH_IF reason="Creating a subdirectory inside a freshly created temporary directory fails only on OS filesystem exhaustion."
	if err := agentrun.CreateDir(tempConfig, 0o700); err != nil {
		return "", noRelease, errors.Join(fmt.Errorf("create %s: %w", tempConfig, err), cleanup())
	}
	if err := agentrun.LinkEntries(realGemini, tempGemini, "config"); err != nil {
		return "", noRelease, errors.Join(err, cleanup())
	}
	if err := agentrun.LinkEntries(realConfig, tempConfig, "mcp_config.json"); err != nil {
		return "", noRelease, errors.Join(err, cleanup())
	}

	mcpConfig := map[string]any{"mcpServers": map[string]any{}}
	if tools != nil {
		mcpConfig = makeMCPConfig(*tools, filepath.Join(realConfig, "mcp_config.json"))
	}
	encoded, err := json.MarshalIndent(mcpConfig, "", "  ")
	// gomutants:disable-next-line BRANCH_IF reason="json.MarshalIndent on a map of decoded JSON values and string maps cannot fail."
	if err != nil {
		return "", noRelease, errors.Join(fmt.Errorf("encode the MCP configuration: %w", err), cleanup())
	}
	configPath := filepath.Join(tempConfig, "mcp_config.json")
	// gomutants:disable-next-line BRANCH_IF reason="Writing a small file inside a freshly created temporary directory fails only on OS filesystem exhaustion."
	if err := agentrun.WriteFile(configPath, append(encoded, '\n'), 0o600); err != nil {
		return "", noRelease, errors.Join(fmt.Errorf("write %s: %w", configPath, err), cleanup())
	}

	return tempHome, cleanup, nil
}

func makeMCPConfig(tools endpoint, existingPath string) map[string]any {
	servers := map[string]any{
		"numen": map[string]any{
			"type":      "http",
			"serverUrl": tools.URL,
			"headers": map[string]string{
				"Authorization": "Bearer " + tools.Token,
			},
		},
	}
	for k, v := range readMCPServers(existingPath) {
		if k != "numen" {
			servers[k] = v
		}
	}
	return map[string]any{"mcpServers": servers}
}

func readMCPServers(path string) map[string]any {
	raw, err := agentrun.ReadFile(path)
	if err != nil {
		return nil
	}
	var existing struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &existing); err != nil {
		return nil
	}
	return existing.MCPServers
}
