package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// CodexConfig holds configuration values discovered from Codex configuration files.
type CodexConfig struct {
	Model      string         `json:"model,omitempty"`
	MCPServers map[string]any `json:"mcp_servers,omitempty"`
}

// ReadCodexConfig reads the OpenAI Codex configuration from standard locations.
func ReadCodexConfig() (*CodexConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	tomlPath := filepath.Join(home, ".codex", "config.toml")
	if data, err := agentrun.ReadFile(tomlPath); err == nil {
		cfg := parseSimpleToml(string(data))
		return cfg, nil
	}
	jsonPath := filepath.Join(home, ".codex", "config.json")
	if data, err := agentrun.ReadFile(jsonPath); err == nil {
		var cfg CodexConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}
	return &CodexConfig{Model: "", MCPServers: nil}, nil
}

func parseSimpleToml(content string) *CodexConfig {
	cfg := &CodexConfig{
		Model:      "",
		MCPServers: make(map[string]any),
	}
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// gomutants:disable-next-line INVERT_LOGICAL reason="comments start with # and empty lines yield a single split part, so neither matches the key 'model' when fallthrough occurs"
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			if k == "model" {
				cfg.Model = v
			}
		}
	}
	return cfg
}
