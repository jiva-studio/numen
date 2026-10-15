// Package plugin manages discovery, lifecycle, and communication with external plugins.
package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

var (
	errEmptyID      = errors.New("plugin id cannot be empty")
	errEmptyName    = errors.New("plugin name cannot be empty")
	errEmptyVersion = errors.New("plugin version cannot be empty")
)

// Lifecycle defines how the host activates and evicts a plugin from memory.
type Lifecycle struct {
	Mode                  string `json:"mode"`
	IdleTimeoutSeconds    int    `json:"idleTimeoutSeconds"`
	KeepAliveWhileVisible bool   `json:"keepAliveWhileVisible"`
}

// Backend describes the server-side executable or wasm module of a plugin.
type Backend struct {
	Type        string `json:"type"`
	Entrypoint  string `json:"entrypoint"`
	ServiceName string `json:"serviceName"`
}

// Frontend describes the client-side bundle and styles of a plugin.
type Frontend struct {
	Entrypoint string `json:"entrypoint"`
	Styles     string `json:"styles"`
}

// SettingOption defines one selectable choice in an enum setting.
type SettingOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Setting declares one configuration field managed by the core host.
type Setting struct {
	Key         string          `json:"key"`
	Type        string          `json:"type"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Default     any             `json:"default"`
	Palette     bool            `json:"palette"`
	Options     []SettingOption `json:"options"`
	Min         float64         `json:"min"`
	Max         float64         `json:"max"`
	Step        float64         `json:"step"`
}

// AgentTool declares a custom tool provided or required by an agent plugin.
type AgentTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AgentConfig declares agent system prompt extensions and custom tools in the manifest.
type AgentConfig struct {
	SystemPrompt string      `json:"systemPrompt"`
	Tools        []AgentTool `json:"tools"`
}

// Manifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type Manifest struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Version          string      `json:"version"`
	Description      string      `json:"description"`
	Capabilities     []string    `json:"capabilities"`
	Targets          []string    `json:"targets"`
	FileExtensions   []string    `json:"fileExtensions,omitempty"`
	ActivationEvents []string    `json:"activationEvents"`
	Permissions      []string    `json:"permissions"`
	Lifecycle        Lifecycle   `json:"lifecycle"`
	Backend          Backend     `json:"backend"`
	Frontend         Frontend    `json:"frontend"`
	Settings         []Setting   `json:"settings"`
	Agent            AgentConfig `json:"agent"`
}

// ReadManifest decodes and validates a manifest from JSON input.
func ReadManifest(r io.Reader) (Manifest, error) {
	var m Manifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return Manifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Validate checks that required fields are present and valid.
func (m Manifest) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return errEmptyID
	}
	if strings.TrimSpace(m.Name) == "" {
		return errEmptyName
	}
	if strings.TrimSpace(m.Version) == "" {
		return errEmptyVersion
	}
	return nil
}

// CanTarget reports whether this plugin runs in the given workspace target.
func (m Manifest) CanTarget(target string) bool {
	if len(m.Targets) == 0 {
		return true
	}
	return slices.Contains(m.Targets, target)
}

// HasCapability reports whether this plugin provides the given capability.
func (m Manifest) HasCapability(capability string) bool {
	return slices.Contains(m.Capabilities, capability)
}

// HasFileExtension reports whether this plugin supports the given file extension.
func (m Manifest) HasFileExtension(ext string) bool {
	return slices.Contains(m.FileExtensions, ext)
}

// CanActivateOn reports whether the given event triggers plugin activation.
func (m Manifest) CanActivateOn(event string) bool {
	if slices.Contains(m.ActivationEvents, "*") {
		return true
	}
	for _, one := range m.ActivationEvents {
		if one == event {
			return true
		}
		if strings.HasPrefix(one, "onFile:") && strings.HasPrefix(event, "onFile:") {
			pattern := strings.TrimPrefix(one, "onFile:")
			target := strings.TrimPrefix(event, "onFile:")
			if matchFilePattern(pattern, target) {
				return true
			}
		}
	}
	return false
}

// matchFilePattern checks if a target file matches a simple extension or wildcard.
func matchFilePattern(pattern, target string) bool {
	if pattern == target || pattern == "*" {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		ext := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(target, ext)
	}
	if strings.HasPrefix(pattern, ".") {
		return strings.HasSuffix(target, pattern)
	}
	return false
}
