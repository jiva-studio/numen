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

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = "vault:read"

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = "vault:write"

	// PermissionNetworkOutbound allows outbound network requests.
	PermissionNetworkOutbound = "network:outbound"
)

var (
	errEmptyID             = errors.New("plugin id cannot be empty")
	errEmptyName           = errors.New("plugin name cannot be empty")
	errEmptyVersion        = errors.New("plugin version cannot be empty")
	errInvalidPermission   = errors.New("invalid permission")
	errEmptySettingKey     = errors.New("setting key cannot be empty")
	errDuplicateSettingKey = errors.New("duplicate setting key")
	errInvalidSettingType  = errors.New("invalid setting type")
	errEnumMissingOptions  = errors.New("enum setting must define options")
	errRangeInvalidBounds  = errors.New("range min cannot exceed max")
)

var validSettingTypes = []string{"boolean", "string", "number", "enum", "select", "range"}

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

// SettingOptions is a slice of option values supporting string arrays and object arrays.
type SettingOptions []string

func (o *SettingOptions) UnmarshalJSON(data []byte) error {
	if string(data) == "null" || len(data) == 0 {
		*o = nil
		return nil
	}
	var stringSlice []string
	if err := json.Unmarshal(data, &stringSlice); err == nil {
		*o = stringSlice
		return nil
	}

	var objSlice []struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &objSlice); err == nil {
		vals := make([]string, len(objSlice))
		for i, obj := range objSlice {
			vals[i] = obj.Value
		}
		*o = vals
		return nil
	}

	return fmt.Errorf("options must be array of strings or objects with value")
}

// Setting declares one configuration field managed by the core host.
type Setting struct {
	Key         string         `json:"key"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Default     any            `json:"default"`
	Palette     bool           `json:"palette,omitempty"`
	Options     SettingOptions `json:"options,omitempty"`
	Min         float64        `json:"min,omitempty"`
	Max         float64        `json:"max,omitempty"`
	Step        float64        `json:"step,omitempty"`
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

// Validate checks that required fields and settings are present and valid.
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
	if m.Backend.Type != "" && m.Backend.Type != "process" {
		return fmt.Errorf("unsupported backend type: %s", m.Backend.Type)
	}
	if err := m.ValidatePermissions(); err != nil {
		return err
	}
	return validateSettings(m.Settings)
}

// ValidatePermissions verifies that all declared permissions are recognized capabilities.
func (m Manifest) ValidatePermissions() error {
	for _, p := range m.Permissions {
		switch p {
		case PermissionVaultRead, PermissionVaultWrite, PermissionNetworkOutbound, "network":
		default:
			return fmt.Errorf("%w: %s", errInvalidPermission, p)
		}
	}
	return nil
}

func validateSettings(settings []Setting) error {
	seenKeys := make(map[string]bool, len(settings))
	for _, s := range settings {
		key := strings.TrimSpace(s.Key)
		if key == "" {
			return errEmptySettingKey
		}
		if seenKeys[key] {
			return fmt.Errorf("%w: %s", errDuplicateSettingKey, key)
		}
		seenKeys[key] = true
		if err := validateSetting(s, key); err != nil {
			return err
		}
	}
	return nil
}

func validateSetting(s Setting, key string) error {
	if !slices.Contains(validSettingTypes, s.Type) {
		return fmt.Errorf("%w: %q for %s", errInvalidSettingType, s.Type, key)
	}
	if (s.Type == "enum" || s.Type == "select") && len(s.Options) == 0 {
		return fmt.Errorf("%w: %s", errEnumMissingOptions, key)
	}
	if s.Type == "range" && (s.Min != 0 || s.Max != 0) && s.Min > s.Max {
		return fmt.Errorf("%w: %s", errRangeInvalidBounds, key)
	}
	return nil
}

// HasPermission reports whether this plugin requests the given permission.
func (m Manifest) HasPermission(permission string) bool {
	if permission == PermissionNetworkOutbound || permission == "network" {
		return slices.Contains(m.Permissions, PermissionNetworkOutbound) || slices.Contains(m.Permissions, "network")
	}
	return slices.Contains(m.Permissions, permission)
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
		rawExt := strings.TrimPrefix(pattern, "*.")
		return strings.HasSuffix(target, ext) || target == rawExt || target == ext
	}
	if strings.HasPrefix(pattern, ".") {
		rawExt := strings.TrimPrefix(pattern, ".")
		return strings.HasSuffix(target, pattern) || target == rawExt
	}
	return false
}
