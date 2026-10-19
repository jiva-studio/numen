package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CodexConfig holds configuration values discovered from Codex configuration files.
type CodexConfig struct {
	Model      string         `json:"model,omitempty"`
	MCPServers map[string]any `json:"mcp_servers,omitempty"`
}

// AntigravityConfig holds configuration values discovered from Antigravity configuration files.
type AntigravityConfig struct {
	Model      string         `json:"model,omitempty"`
	MCPServers map[string]any `json:"mcpServers,omitempty"`
}

// ReadCodexConfig reads the OpenAI Codex configuration from standard locations.
func ReadCodexConfig() (*CodexConfig, error) {
	home, err := os.UserHomeDir()
	// gomutants:disable-next-line BRANCH_IF reason="os.UserHomeDir fails only when user home directory cannot be determined"
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
	return &CodexConfig{}, nil
}

// ReadAntigravityConfig reads the Google Antigravity CLI configuration from standard locations.
func ReadAntigravityConfig() (*AntigravityConfig, error) {
	home, err := os.UserHomeDir()
	// gomutants:disable-next-line BRANCH_IF reason="os.UserHomeDir fails only when user home directory cannot be determined"
	if err != nil {
		return nil, err
	}
	candidates := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "config.json"),
		filepath.Join(home, ".gemini", "config.json"),
		filepath.Join(home, ".antigravity", "config.json"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg AntigravityConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}
	return &AntigravityConfig{}, nil
}

func parseSimpleToml(content string) *CodexConfig {
	cfg := &CodexConfig{
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
