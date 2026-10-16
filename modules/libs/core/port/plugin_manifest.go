package port

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
	ErrEmptyID             = errors.New("plugin id cannot be empty")
	ErrEmptyName           = errors.New("plugin name cannot be empty")
	ErrEmptyVersion        = errors.New("plugin version cannot be empty")
	ErrInvalidPermission   = errors.New("invalid permission")
	ErrEmptySettingKey     = errors.New("setting key cannot be empty")
	ErrDuplicateSettingKey = errors.New("duplicate setting key")
	ErrInvalidSettingType  = errors.New("invalid setting type")
	ErrEnumMissingOptions  = errors.New("enum setting must define options")
	ErrRangeInvalidBounds  = errors.New("range min cannot exceed max")
)

var validSettingTypes = []string{"boolean", "string", "number", "enum", "select", "range"}

// PluginLifecycle defines how the host activates and evicts a plugin from memory.
type PluginLifecycle struct {
	Mode                  string `json:"mode"`
	IdleTimeoutSeconds    int    `json:"idleTimeoutSeconds"`
	KeepAliveWhileVisible bool   `json:"keepAliveWhileVisible"`
}

// PluginBackend describes the server-side executable or wasm module of a plugin.
type PluginBackend struct {
	Type        string `json:"type"`
	Entrypoint  string `json:"entrypoint"`
	ServiceName string `json:"serviceName"`
}

// PluginFrontend describes the client-side bundle and styles of a plugin.
type PluginFrontend struct {
	Entrypoint string `json:"entrypoint"`
	Styles     string `json:"styles"`
}

// PluginSettingOptions is a slice of option values supporting string arrays and object arrays.
type PluginSettingOptions []string

// UnmarshalJSON unmarshals either string array or object array with value field.
func (o *PluginSettingOptions) UnmarshalJSON(data []byte) error {
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

// PluginManifestSetting declares one configuration field managed by the core host.
type PluginManifestSetting struct {
	Key         string               `json:"key"`
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Default     any                  `json:"default"`
	Palette     bool                 `json:"palette,omitempty"`
	Options     PluginSettingOptions `json:"options,omitempty"`
	Min         float64              `json:"min,omitempty"`
	Max         float64              `json:"max,omitempty"`
	Step        float64              `json:"step,omitempty"`
}

// PluginAgentTool declares a custom tool provided or required by an agent plugin.
type PluginAgentTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PluginAgentConfig declares agent system prompt extensions and custom tools in the manifest.
type PluginAgentConfig struct {
	SystemPrompt string            `json:"systemPrompt"`
	Tools        []PluginAgentTool `json:"tools"`
}

// PluginManifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type PluginManifest struct {
	ID               string                  `json:"id"`
	Name             string                  `json:"name"`
	Version          string                  `json:"version"`
	Description      string                  `json:"description"`
	Capabilities     []string                `json:"capabilities"`
	Targets          []string                `json:"targets"`
	FileExtensions   []string                `json:"fileExtensions,omitempty"`
	ActivationEvents []string                `json:"activationEvents"`
	Permissions      []string                `json:"permissions"`
	Lifecycle        PluginLifecycle         `json:"lifecycle"`
	Backend          PluginBackend           `json:"backend"`
	Frontend         PluginFrontend          `json:"frontend"`
	Settings         []PluginManifestSetting `json:"settings"`
	Agent            PluginAgentConfig       `json:"agent"`
}

// ReadManifest decodes and validates a manifest from JSON input.
func ReadManifest(r io.Reader) (PluginManifest, error) {
	var m PluginManifest
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		return PluginManifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return PluginManifest{}, err
	}
	return m, nil
}

// Validate checks that required fields and settings are present and valid.
func (m PluginManifest) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return ErrEmptyID
	}
	if strings.TrimSpace(m.Name) == "" {
		return ErrEmptyName
	}
	if strings.TrimSpace(m.Version) == "" {
		return ErrEmptyVersion
	}
	if m.Backend.Type != "" && m.Backend.Type != "process" {
		return fmt.Errorf("unsupported backend type: %s", m.Backend.Type)
	}
	if err := m.ValidatePermissions(); err != nil {
		return err
	}
	return validateManifestSettings(m.Settings)
}

// ValidatePermissions verifies that all declared permissions are recognized capabilities.
func (m PluginManifest) ValidatePermissions() error {
	for _, p := range m.Permissions {
		switch p {
		case PermissionVaultRead, PermissionVaultWrite, PermissionNetworkOutbound, "network":
		default:
			return fmt.Errorf("%w: %s", ErrInvalidPermission, p)
		}
	}
	return nil
}

func validateManifestSettings(settings []PluginManifestSetting) error {
	seenKeys := make(map[string]bool, len(settings))
	for _, s := range settings {
		key := strings.TrimSpace(s.Key)
		if key == "" {
			return ErrEmptySettingKey
		}
		if seenKeys[key] {
			return fmt.Errorf("%w: %s", ErrDuplicateSettingKey, key)
		}
		seenKeys[key] = true
		if err := validateManifestSetting(s, key); err != nil {
			return err
		}
	}
	return nil
}

func validateManifestSetting(s PluginManifestSetting, key string) error {
	if !slices.Contains(validSettingTypes, s.Type) {
		return fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key)
	}
	if (s.Type == "enum" || s.Type == "select") && len(s.Options) == 0 {
		return fmt.Errorf("%w: %s", ErrEnumMissingOptions, key)
	}
	if s.Type == "range" && (s.Min != 0 || s.Max != 0) && s.Min > s.Max {
		return fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key)
	}
	return nil
}

// HasPermission reports whether this plugin requests the given permission.
func (m PluginManifest) HasPermission(permission string) bool {
	if permission == PermissionNetworkOutbound || permission == "network" {
		return slices.Contains(m.Permissions, PermissionNetworkOutbound) || slices.Contains(m.Permissions, "network")
	}
	return slices.Contains(m.Permissions, permission)
}

// CanTarget reports whether this plugin runs in the given workspace target.
func (m PluginManifest) CanTarget(target string) bool {
	if len(m.Targets) == 0 {
		return true
	}
	return slices.Contains(m.Targets, target)
}

// HasCapability reports whether this plugin provides the given capability.
func (m PluginManifest) HasCapability(capability string) bool {
	return slices.Contains(m.Capabilities, capability)
}

// HasFileExtension reports whether this plugin supports the given file extension.
func (m PluginManifest) HasFileExtension(ext string) bool {
	return slices.Contains(m.FileExtensions, ext)
}

// CanActivateOn reports whether the given event triggers plugin activation.
func (m PluginManifest) CanActivateOn(event string) bool {
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
