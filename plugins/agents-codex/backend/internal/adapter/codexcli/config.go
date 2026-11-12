package codexcli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CodexConfig holds configuration values discovered from Codex configuration files.
type CodexConfig struct {
	Model      string         `json:"model,omitempty"`
	MCPServers map[string]any `json:"mcpServers,omitempty"`
}

// ReadCodexConfig reads the OpenAI Codex configuration from standard locations.
func ReadCodexConfig() (*CodexConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	tomlPath := filepath.Join(home, ".codex", "config.toml")
	if data, err := os.ReadFile(tomlPath); err == nil {
		cfg := parseSimpleToml(string(data))
		return cfg, nil
	}
	jsonPath := filepath.Join(home, ".codex", "config.json")
	if data, err := os.ReadFile(jsonPath); err == nil {
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
	lines := strings.SplitSeq(content, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// gomutants:disable-next-line INVERT_LOGICAL,REMOVE_CONDITIONAL,BRANCH_IF reason="Comments start with # and empty lines contain no = separator, so falling through to SplitN produces no model key match."
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
