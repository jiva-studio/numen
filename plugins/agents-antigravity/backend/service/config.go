// Package service implements the Antigravity agent plugin backend.
package service

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/agent/agentrun"
)

// AntigravityConfig holds configuration values discovered from Antigravity configuration files.
type AntigravityConfig struct {
	Model      string         `json:"model,omitempty"`
	MCPServers map[string]any `json:"mcpServers,omitempty"`
}

// ReadAntigravityConfig reads the Google Antigravity CLI configuration from standard locations.
func ReadAntigravityConfig() (*AntigravityConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	candidates := []string{
		filepath.Join(home, ".gemini", "antigravity-cli", "config.json"),
		filepath.Join(home, ".gemini", "config.json"),
		filepath.Join(home, ".antigravity", "config.json"),
	}
	for _, path := range candidates {
		data, err := agentrun.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg AntigravityConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return &cfg, nil
		}
	}
	return &AntigravityConfig{Model: "", MCPServers: nil}, nil
}
