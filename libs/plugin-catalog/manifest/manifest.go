package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
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

	// PermissionIndexRead allows reading the search index of a vault: its note titles, the stored
	// vectors of its chunks, and the state of its embedding.
	PermissionIndexRead = "index:read"

	// PermissionSearchPolicy allows reading and setting the floor that a search by meaning applies.
	PermissionSearchPolicy = "search:policy"

	// PermissionNetworkAlias is the accepted short spelling of PermissionNetworkOutbound.
	PermissionNetworkAlias = "network"

	ProtocolVersion = 1
)

// PluginPermissions is every permission a manifest may declare, in its canonical spelling.
func PluginPermissions() []string {
	return []string{
		PermissionVaultRead, PermissionVaultWrite, PermissionNetworkOutbound, PermissionAIComplete,
		PermissionIndexRead, PermissionSearchPolicy,
	}
}

// ActivationEventAll is the activation event that matches every event the host emits.
const ActivationEventAll = "*"

// PluginCapabilities is every capability a manifest may declare: the capabilities the host resolves
// a plugin by.
func PluginCapabilities() []string {
	return []string{
		"embedding",
		"tools",
		"ai.agent.provider",
	}
}

var (
	ErrEmptyID                = errors.New("plugin id cannot be empty")
	ErrEmptyName              = errors.New("plugin name cannot be empty")
	ErrEmptyVersion           = errors.New("plugin version cannot be empty")
	ErrInvalidID              = errors.New("plugin id must be reverse-DNS: two or more dot-separated segments of letters, digits, _ or - (e.g. org.example.sample)")
	ErrInvalidVersion         = errors.New("plugin version is not valid semver (e.g. 1.0.0)")
	ErrUnsupportedProtocol    = errors.New("unsupported plugin protocol")
	ErrEmptySettingTitle      = errors.New("setting title cannot be empty")
	ErrInvalidBackend         = errors.New("invalid backend")
	ErrInvalidPermission      = errors.New("invalid permission")
	ErrInvalidCapability      = errors.New("invalid capability")
	ErrInvalidActivationEvent = errors.New("invalid activation event")
	ErrEmptySettingKey        = errors.New("setting key cannot be empty")
	ErrInvalidSettingKey      = errors.New("invalid setting key")
	ErrDuplicateSettingKey    = errors.New("duplicate setting key")
	ErrInvalidSettingType     = errors.New("invalid setting type")
	ErrEnumMissingOptions     = errors.New("enum setting must define options")
	ErrRangeInvalidBounds     = errors.New("range min cannot exceed max")
	ErrInvalidSecret          = errors.New("invalid secret")
	ErrReservedSettingKey     = errors.New("the setting key is kept by the host")
	ErrInvalidSetting         = errors.New("invalid setting")
	ErrInvalidService         = errors.New("invalid service name")
	ErrInvalidNoteType        = errors.New("invalid note type")
	ErrNoBackend              = errors.New("only a backend serves this, and the plugin declares none")
)

// SettingStoredSecrets is the setting under which the host records, for a plugin,
// the names of its secrets that have a key in the key store. It holds names and
// never a key, and neither a setting nor a secret of a manifest takes the name.
const SettingStoredSecrets = "storedSecrets"

// SettingAIConsent is the setting under which the host records AI consent;
// absent means the user has not been prompted yet. The window writes it, and no
// manifest uses the name.
const SettingAIConsent = "aiConsent"

// reservedSettingKeys are the settings of a plugin the host keeps for itself.
var reservedSettingKeys = []string{SettingStoredSecrets, SettingAIConsent}

var (
	manifestIDPattern       = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	manifestServicePattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
	manifestReplacesPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	manifestSecretPattern   = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	manifestVersionPattern  = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))(\.((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)))*)?(\+([0-9A-Za-z-]+)(\.[0-9A-Za-z-]+)*)?$`)
)

// SettingTypeString is the type of a setting that holds a text string.
const SettingTypeString = "string"

// SettingTypeEnum is the type of a setting that holds one of its options.
const SettingTypeEnum = "enum"

// SettingTypeFile is the type of a setting that holds a file path on this machine.
const SettingTypeFile = "file"

var validSettingTypes = []string{"boolean", SettingTypeString, "number", SettingTypeEnum, "range", SettingTypeFile}

// Backend describes the server-side executable of a plugin.
type Backend struct {
	Entrypoint string
}

// Frontend declares that a plugin has a client-side bundle. The bundle is dist/index.js and its
// stylesheet is dist/styles.css in the plugin folder.
type Frontend struct{}

// Setting declares one configuration field managed by the core host.
type Setting struct {
	Key         string
	Type        string
	Title       string
	Description string
	Default     any
	Palette     bool
	Icon        string
	Options     SettingOptions
	// Min, Max and Step are nil where the manifest sets none.
	Min  *float64
	Max  *float64
	Step *float64
	// Replaces are the places in the settings file where the user stored this setting before the plugin took
	// it over, each written as the dot-joined keys leading to it, the newest first. The first of
	// them the file contains supplies the value for a user whose file has none for the plugin yet.
	Replaces []string
	// OptionsBy gives an enum setting its options from the value of another setting. The first option
	// of each list is the default under that value.
	OptionsBy *OptionsBy
	// When is the condition under which the setting is shown and handed to the plugin. A setting
	// whose condition fails keeps its stored value.
	When *Condition
	// Accept are the file extensions a setting of type file takes, each with its leading dot. Empty
	// takes any file.
	Accept []string
}

// Manifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type Manifest struct {
	ID               string
	Name             string
	Version          string
	Protocol         int
	Description      string
	Capabilities     []string
	Targets          []string
	ActivationEvents []string
	Permissions      []string
	Dependencies     []DependencyRequirement
	Backend          Backend
	// Frontend is nil for a plugin without a frontend.
	Frontend *Frontend
	Settings []Setting
	// Secrets are the names of the secrets, such as an API key, that the plugin
	// requests the host to store. The host passes them as Configure values,
	// empty while a secret has no key, and never puts one in the environment or
	// on the command line.
	Secrets []Secret
	// Services are the services the plugin serves to the windows, by their full names. A window that
	// receives a call for one forwards it to the plugin; a service the host serves itself is never
	// forwarded.
	Services []string
	// Commands are the commands the plugin's backend runs, offered by the palette and the command line.
	Commands []Command
	// SettingsViews are the components of the plugin's frontend the settings page draws among the
	// settings of the plugin, each by the id the frontend registers it under.
	SettingsViews []SettingsView
	// Contributes is what the plugin adds to the application, known before its frontend is loaded.
	Contributes Contributes
	// Warnings are the findings of the reader that do not make the manifest invalid, such as a key the
	// schema does not name. The reader fills them and a manifest built in code has none.
	Warnings []string
}

// NoteType declares one type of note and how the index keeps the notes of it.
type NoteType struct {
	// Type is the word the `type` key of a note's frontmatter holds.
	Type string
	// Chunks states whether the prose of the note is split into chunks and
	// searched by meaning. When omitted, it is.
	Chunks *bool
	// Headings are the levels of heading the index keeps. Left out, every heading is kept; an empty
	// list keeps none.
	Headings *[]int
	// IDHeadingLevels are the levels whose headings end in a block id (` ^id`, as in Obsidian) that
	// is no part of their text.
	IDHeadingLevels []int
	// IDPattern is a regular expression a block id must match whole. Left out, any word of letters,
	// digits and hyphens is an id.
	IDPattern string
}

// GetProtocol returns the plugin protocol the manifest declares. An absent protocol is 1.
func (m Manifest) GetProtocol() int {
	if m.Protocol == 0 {
		return 1
	}
	return m.Protocol
}

// Validate checks the manifest and returns every violation joined into one error.
func (m Manifest) Validate() error {
	return errors.Join(m.ListIssues()...)
}

// ListIssues returns every rule the manifest breaks, one error each.
func (m Manifest) ListIssues() []error {
	var issues []error
	issues = append(issues, m.listIdentityIssues()...)
	if m.Protocol < 0 || m.Protocol > ProtocolVersion {
		issues = append(issues, fmt.Errorf("%w: %d, this host speaks up to %d", ErrUnsupportedProtocol, m.Protocol, ProtocolVersion))
	}
	issues = append(issues, m.listCapabilityIssues()...)
	issues = append(issues, m.listActivationEventIssues()...)
	issues = append(issues, m.listEntrypointIssues()...)
	issues = append(issues, m.listBackendlessIssues()...)
	issues = append(issues, m.listPermissionIssues()...)
	issues = append(issues, m.listDependencyIssues()...)
	issues = append(issues, m.listSecretIssues()...)
	issues = append(issues, m.listServiceIssues()...)
	issues = append(issues, m.listNoteTypeIssues()...)
	issues = append(issues, m.listCommandIssues()...)
	issues = append(issues, m.listSettingsViewIssues()...)
	issues = append(issues, m.listContributionIssues()...)
	issues = append(issues, m.listSettingLinkIssues()...)
	return append(issues, listManifestSettingIssues(m.Settings)...)
}

// listServiceIssues checks that each service is a full name of dot-separated words, and is listed once.
func (m Manifest) listServiceIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, name := range m.Services {
		switch {
		case !manifestServicePattern.MatchString(name):
			issues = append(issues, fmt.Errorf("%w: %q", ErrInvalidService, name))
		case observed[name]:
			issues = append(issues, fmt.Errorf("%w: %q is listed twice", ErrInvalidService, name))
		}
		observed[name] = true
	}
	return issues
}

// listNoteTypeIssues checks that each type is a word the frontmatter can hold, is declared once, and
// that the levels it names are those of headings.
func (m Manifest) listNoteTypeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range m.Contributes.NoteTypes {
		switch {
		case !manifestSecretPattern.MatchString(declared.Type):
			issues = append(issues, fmt.Errorf("%w: %q", ErrInvalidNoteType, declared.Type))
		case observed[declared.Type]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidNoteType, declared.Type))
		}
		observed[declared.Type] = true
		if _, err := regexp.Compile(declared.IDPattern); err != nil {
			issues = append(issues, fmt.Errorf("%w: %q has an id pattern that is no pattern: %w", ErrInvalidNoteType, declared.Type, err))
		}
		levels := declared.IDHeadingLevels
		if declared.Headings != nil {
			levels = append(slices.Clone(*declared.Headings), levels...)
		}
		for _, level := range levels {
			if level < 1 || level > 6 {
				issues = append(issues, fmt.Errorf("%w: %q names heading level %d", ErrInvalidNoteType, declared.Type, level))
			}
		}
	}
	return issues
}

// GetProfile is how the index keeps the notes of the type.
func (t NoteType) GetProfile() domain.NoteProfile {
	profile := domain.DefaultProfile()
	if t.Chunks != nil {
		profile.HasChunks = *t.Chunks
	}
	if t.Headings != nil {
		profile.HeadingLevels = slices.Clone(*t.Headings)
		if profile.HeadingLevels == nil {
			profile.HeadingLevels = []int{}
		}
	}
	profile.IDHeadingLevels = slices.Clone(t.IDHeadingLevels)
	if t.IDPattern != "" {
		// listNoteTypeIssues rejects a pattern that does not compile.
		if pattern, err := regexp.Compile("^(?:" + t.IDPattern + ")$"); err == nil {
			profile.IDPattern = pattern
		}
	}
	return profile
}

// GetNoteType is the declaration of a type of note the plugin gives meaning to.
func (m Manifest) GetNoteType(noteType string) (NoteType, bool) {
	for _, declared := range m.Contributes.NoteTypes {
		if declared.Type == noteType {
			return declared, true
		}
	}
	return NoteType{}, false
}

func (m Manifest) listIdentityIssues() []error {
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

func (m Manifest) listEntrypointIssues() []error {
	if m.Backend != (Backend{}) && strings.TrimSpace(m.Backend.Entrypoint) == "" {
		return []error{fmt.Errorf("%w: entrypoint cannot be empty", ErrInvalidBackend)}
	}
	return nil
}

// HasBackend reports whether the plugin declares a process the host starts.
func (m Manifest) HasBackend() bool {
	return strings.TrimSpace(m.Backend.Entrypoint) != ""
}

// HasFrontend reports whether the plugin declares a bundle the windows load.
func (m Manifest) HasFrontend() bool {
	return m.Frontend != nil
}

// listBackendlessIssues names each field of a plugin without a backend that only a backend serves:
// the host would start a process to answer it.
func (m Manifest) listBackendlessIssues() []error {
	if m.Backend != (Backend{}) {
		return nil
	}
	fields := []struct {
		name       string
		isDeclared bool
	}{
		{"activationEvents", len(m.ActivationEvents) > 0},
		{"commands", len(m.Commands) > 0},
		{"services", len(m.Services) > 0},
		{"secrets", len(m.Secrets) > 0},
		{"runtime dependencies", slices.ContainsFunc(m.Dependencies, func(d DependencyRequirement) bool { return d.Kind == DependencyKindRuntime })},
		{"contributes.handles", len(m.Contributes.Handles) > 0},
	}
	var issues []error
	for _, field := range fields {
		if field.isDeclared {
			issues = append(issues, fmt.Errorf("%w: %s", ErrNoBackend, field.name))
		}
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
func (m Manifest) ValidatePermissions() error {
	return errors.Join(m.listPermissionIssues()...)
}

func (m Manifest) listPermissionIssues() []error {
	var issues []error
	for _, p := range m.Permissions {
		if !slices.Contains(PluginPermissions(), NormalizePermission(p)) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidPermission, p))
		}
	}
	return issues
}

// listCapabilityIssues checks that every capability is one the host resolves a plugin by.
func (m Manifest) listCapabilityIssues() []error {
	var issues []error
	for _, c := range m.Capabilities {
		if !slices.Contains(PluginCapabilities(), c) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidCapability, c))
		}
	}
	return issues
}

// listActivationEventIssues checks that every activation event is ActivationEventAll.
func (m Manifest) listActivationEventIssues() []error {
	var issues []error
	for _, e := range m.ActivationEvents {
		if e != ActivationEventAll {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidActivationEvent, e))
		}
	}
	return issues
}

func (m Manifest) listSecretIssues() []error {
	var issues []error
	observed := make(map[string]bool, len(m.Secrets))
	for _, secret := range m.Secrets {
		name := secret.Name
		issues = append(issues, m.listConditionIssues("secret "+name, secret.When)...)
		switch {
		case !manifestSecretPattern.MatchString(name):
			issues = append(issues, fmt.Errorf("%w: %q is not letters, digits, _ or -", ErrInvalidSecret, name))
		case slices.Contains(reservedSettingKeys, name):
			issues = append(issues, fmt.Errorf("%w: secret %q", ErrReservedSettingKey, name))
		case observed[name]:
			issues = append(issues, fmt.Errorf("%w: %q is declared twice", ErrInvalidSecret, name))
		case slices.ContainsFunc(m.Settings, func(s Setting) bool { return s.Key == name }):
			issues = append(issues, fmt.Errorf("%w: %q is also a setting", ErrInvalidSecret, name))
		}
		observed[name] = true
	}
	return issues
}

func listManifestSettingIssues(settings []Setting) []error {
	var issues []error
	seenKeys := make(map[string]bool, len(settings))
	for _, s := range settings {
		key := strings.TrimSpace(s.Key)
		if key == "" {
			issues = append(issues, ErrEmptySettingKey)
			continue
		}
		if slices.Contains(reservedSettingKeys, key) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrReservedSettingKey, key))
			continue
		}
		if seenKeys[key] {
			issues = append(issues, fmt.Errorf("%w: %s", ErrDuplicateSettingKey, key))
			continue
		}
		seenKeys[key] = true
		issues = append(issues, listManifestSettingIssue(s, key)...)
		for _, place := range s.Replaces {
			if !manifestReplacesPattern.MatchString(place) {
				issues = append(issues, fmt.Errorf("%w: %s replaces %q", ErrInvalidSettingKey, key, place))
			}
		}
	}
	return issues
}

func listManifestSettingIssue(s Setting, key string) []error {
	var issues []error
	if strings.TrimSpace(s.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEmptySettingTitle, key))
	}
	if !slices.Contains(validSettingTypes, s.Type) {
		issues = append(issues, fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key))
	}
	if s.Type == SettingTypeEnum && len(s.Options) == 0 && s.OptionsBy == nil {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEnumMissingOptions, key))
	}
	if s.Type == "range" && s.Min != nil && s.Max != nil && *s.Min > *s.Max {
		issues = append(issues, fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key))
	}
	return append(issues, listAcceptIssues(s, key)...)
}

// listAcceptIssues checks that only a file setting accepts extensions, each with its leading dot.
func listAcceptIssues(s Setting, key string) []error {
	var issues []error
	if len(s.Accept) > 0 && s.Type != SettingTypeFile {
		issues = append(issues, fmt.Errorf("%w: %s accepts extensions and holds no file", ErrInvalidSetting, key))
	}
	for _, extension := range s.Accept {
		if len(extension) < 2 || !strings.HasPrefix(extension, ".") {
			issues = append(issues, fmt.Errorf("%w: %s accepts %q, which is no extension", ErrInvalidSetting, key, extension))
		}
	}
	return issues
}

// HasPermission reports whether this plugin requests the given permission.
func (m Manifest) HasPermission(permission string) bool {
	want := NormalizePermission(permission)
	return slices.ContainsFunc(m.Permissions, func(p string) bool { return NormalizePermission(p) == want })
}

// CanTarget reports whether this plugin runs in the given workspace target.
func (m Manifest) CanTarget(target string) bool {
	if len(m.Targets) == 0 {
		return true
	}
	return slices.Contains(m.Targets, target)
}

// Supports reports whether this plugin runs in the given workspace target.
func (m Manifest) Supports(target string) bool {
	return m.CanTarget(target)
}

// HasCapability reports whether this plugin provides the given capability.
func (m Manifest) HasCapability(capability string) bool {
	return slices.Contains(m.Capabilities, capability)
}

// HasService reports whether this plugin serves the service to the windows.
func (m Manifest) HasService(service string) bool {
	if service == "" {
		return false
	}
	return slices.Contains(m.Services, service)
}

// IsActivatedByAnyEvent reports whether the plugin declares the activation event `*`: the host
// starts it for every event it emits and delivers the event to it.
func (m Manifest) IsActivatedByAnyEvent() bool {
	return slices.Contains(m.ActivationEvents, ActivationEventAll)
}
