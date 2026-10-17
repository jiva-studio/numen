package port

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
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

	// PermissionAIComplete allows asking the host for a model completion.
	PermissionAIComplete = "ai:complete"

	// PermissionNetworkAlias is the accepted short spelling of PermissionNetworkOutbound.
	PermissionNetworkAlias = "network"

	// RuntimeONNX names the ONNX Runtime library, which the host provisions for a plugin that declares it.
	RuntimeONNX = "onnxruntime"

	// PluginProtocolVersion is the newest plugin protocol this host speaks.
	PluginProtocolVersion = 1
)

var (
	ErrEmptyID             = errors.New("plugin id cannot be empty")
	ErrEmptyName           = errors.New("plugin name cannot be empty")
	ErrEmptyVersion        = errors.New("plugin version cannot be empty")
	ErrInvalidID           = errors.New("plugin id must be reverse-DNS: two or more dot-separated segments of letters, digits, _ or - (e.g. md.numen.sample)")
	ErrInvalidVersion      = errors.New("plugin version is not valid semver (e.g. 1.0.0)")
	ErrUnsupportedProtocol = errors.New("unsupported plugin protocol")
	ErrInvalidLifecycle    = errors.New("invalid lifecycle")
	ErrEmptySettingTitle   = errors.New("setting title cannot be empty")
	ErrInvalidBackend      = errors.New("invalid backend")
	ErrInvalidFrontend     = errors.New("invalid frontend")
	ErrInvalidPermission   = errors.New("invalid permission")
	ErrInvalidRuntime      = errors.New("invalid runtime")
	ErrEmptySettingKey     = errors.New("setting key cannot be empty")
	ErrDuplicateSettingKey = errors.New("duplicate setting key")
	ErrInvalidSettingType  = errors.New("invalid setting type")
	ErrEnumMissingOptions  = errors.New("enum setting must define options")
	ErrRangeInvalidBounds  = errors.New("range min cannot exceed max")
)

var (
	manifestIDPattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	manifestVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))(\.((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)))*)?(\+([0-9A-Za-z-]+)(\.[0-9A-Za-z-]+)*)?$`)
)

var validRuntimes = []string{RuntimeONNX}

var validSettingTypes = []string{"boolean", "string", "number", "enum", "select", "range"}

// PluginLifecycle defines when the host evicts an idle plugin from memory.
type PluginLifecycle struct {
	IdleTimeoutSeconds    int    `json:"idleTimeoutSeconds"`
	KeepAliveWhileVisible bool   `json:"keepAliveWhileVisible"`
	MemoryLimitBytes      uint64 `json:"memoryLimitBytes,omitempty"`
}

// UnmarshalJSON decodes lifecycle settings, accepting memoryLimitBytes or memoryLimitMB.
func (l *PluginLifecycle) UnmarshalJSON(data []byte) error {
	type rawLifecycle PluginLifecycle
	var raw struct {
		rawLifecycle
		MemoryLimitMB uint64 `json:"memoryLimitMB,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*l = PluginLifecycle(raw.rawLifecycle)
	if l.MemoryLimitBytes == 0 && raw.MemoryLimitMB > 0 {
		if raw.MemoryLimitMB > math.MaxUint64/(1024*1024) {
			return errors.New("memory limit in megabytes exceeds maximum value")
		}
		l.MemoryLimitBytes = raw.MemoryLimitMB * 1024 * 1024
	}
	return nil
}

// PluginBackend describes the server-side executable of a plugin.
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
	Protocol         int                     `json:"protocol,omitempty"`
	Description      string                  `json:"description"`
	Capabilities     []string                `json:"capabilities"`
	Targets          []string                `json:"targets"`
	FileExtensions   []string                `json:"fileExtensions,omitempty"`
	ActivationEvents []string                `json:"activationEvents"`
	Permissions      []string                `json:"permissions"`
	Runtimes         []string                `json:"runtimes,omitempty"`
	Lifecycle        PluginLifecycle         `json:"lifecycle"`
	Backend          PluginBackend           `json:"backend,omitzero"`
	Frontend         PluginFrontend          `json:"frontend,omitzero"`
	Settings         []PluginManifestSetting `json:"settings"`
	Agent            PluginAgentConfig       `json:"agent,omitzero"`
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

// Validate checks the manifest and returns every violation joined into one error.
func (m PluginManifest) Validate() error {
	return errors.Join(m.ListIssues()...)
}

// ListIssues returns every rule the manifest breaks, one error each.
func (m PluginManifest) ListIssues() []error {
	var issues []error
	issues = append(issues, m.listIdentityIssues()...)
	if m.Protocol < 0 || m.Protocol > PluginProtocolVersion {
		issues = append(issues, fmt.Errorf("%w: %d, this host speaks up to %d", ErrUnsupportedProtocol, m.Protocol, PluginProtocolVersion))
	}
	if m.Lifecycle.IdleTimeoutSeconds < 0 {
		issues = append(issues, fmt.Errorf("%w: idleTimeoutSeconds cannot be negative", ErrInvalidLifecycle))
	}
	issues = append(issues, m.listEntrypointIssues()...)
	issues = append(issues, m.listPermissionIssues()...)
	issues = append(issues, m.listRuntimeIssues()...)
	return append(issues, listManifestSettingIssues(m.Settings)...)
}

func (m PluginManifest) listIdentityIssues() []error {
	var issues []error
	switch {
	case strings.TrimSpace(m.ID) == "":
		issues = append(issues, ErrEmptyID)
	case !manifestIDPattern.MatchString(m.ID):
		issues = append(issues, fmt.Errorf("%w: %q", ErrInvalidID, m.ID))
	}
	if strings.TrimSpace(m.Name) == "" {
		issues = append(issues, ErrEmptyName)
	}
	switch {
	case strings.TrimSpace(m.Version) == "":
		issues = append(issues, ErrEmptyVersion)
	case !manifestVersionPattern.MatchString(m.Version):
		issues = append(issues, fmt.Errorf("%w: %q", ErrInvalidVersion, m.Version))
	}
	return issues
}

func (m PluginManifest) listEntrypointIssues() []error {
	var issues []error
	if m.Backend != (PluginBackend{}) {
		if m.Backend.Type != "" && m.Backend.Type != "process" {
			issues = append(issues, fmt.Errorf("%w: type %q must be \"process\" or empty", ErrInvalidBackend, m.Backend.Type))
		}
		if strings.TrimSpace(m.Backend.Entrypoint) == "" {
			issues = append(issues, fmt.Errorf("%w: entrypoint cannot be empty", ErrInvalidBackend))
		}
	}
	if m.Frontend != (PluginFrontend{}) && strings.TrimSpace(m.Frontend.Entrypoint) == "" {
		issues = append(issues, fmt.Errorf("%w: entrypoint cannot be empty", ErrInvalidFrontend))
	}
	return issues
}

// NormalizePermission maps the accepted permission spellings to their canonical form.
func NormalizePermission(permission string) string {
	if permission == PermissionNetworkAlias {
		return PermissionNetworkOutbound
	}
	return permission
}

// ValidatePermissions verifies that all declared permissions are recognized capabilities.
func (m PluginManifest) ValidatePermissions() error {
	return errors.Join(m.listPermissionIssues()...)
}

func (m PluginManifest) listPermissionIssues() []error {
	var issues []error
	for _, p := range m.Permissions {
		switch NormalizePermission(p) {
		case PermissionVaultRead, PermissionVaultWrite, PermissionNetworkOutbound, PermissionAIComplete:
		default:
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidPermission, p))
		}
	}
	return issues
}

func (m PluginManifest) listRuntimeIssues() []error {
	var issues []error
	seen := make(map[string]bool, len(m.Runtimes))
	for _, name := range m.Runtimes {
		switch {
		case !slices.Contains(validRuntimes, name):
			issues = append(issues, fmt.Errorf("%w: %q, expected one of %s", ErrInvalidRuntime, name, strings.Join(validRuntimes, ", ")))
		case seen[name]:
			issues = append(issues, fmt.Errorf("%w: %s is declared twice", ErrInvalidRuntime, name))
		}
		seen[name] = true
	}
	return issues
}

func listManifestSettingIssues(settings []PluginManifestSetting) []error {
	var issues []error
	seenKeys := make(map[string]bool, len(settings))
	for _, s := range settings {
		key := strings.TrimSpace(s.Key)
		if key == "" {
			issues = append(issues, ErrEmptySettingKey)
			continue
		}
		if seenKeys[key] {
			issues = append(issues, fmt.Errorf("%w: %s", ErrDuplicateSettingKey, key))
			continue
		}
		seenKeys[key] = true
		issues = append(issues, listManifestSettingIssue(s, key)...)
	}
	return issues
}

func listManifestSettingIssue(s PluginManifestSetting, key string) []error {
	var issues []error
	if strings.TrimSpace(s.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEmptySettingTitle, key))
	}
	if !slices.Contains(validSettingTypes, s.Type) {
		issues = append(issues, fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key))
	}
	if (s.Type == "enum" || s.Type == "select") && len(s.Options) == 0 {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEnumMissingOptions, key))
	}
	if s.Type == "range" && (s.Min != 0 || s.Max != 0) && s.Min > s.Max {
		issues = append(issues, fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key))
	}
	return issues
}

// HasPermission reports whether this plugin requests the given permission.
func (m PluginManifest) HasPermission(permission string) bool {
	want := NormalizePermission(permission)
	return slices.ContainsFunc(m.Permissions, func(p string) bool { return NormalizePermission(p) == want })
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
