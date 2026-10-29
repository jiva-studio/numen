package pluginport

import (
	"errors"
	"fmt"
	"io"
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

	// PermissionNetworkAlias is the accepted short spelling of PermissionNetworkOutbound.
	PermissionNetworkAlias = "network"

	// RuntimeONNX names the ONNX Runtime library, which the host provisions for a plugin that declares it.
	RuntimeONNX = "onnxruntime"

	PluginProtocolVersion = 1
)

// PluginPermissions is every permission a manifest may declare, in its canonical spelling.
func PluginPermissions() []string {
	return []string{PermissionVaultRead, PermissionVaultWrite, PermissionNetworkOutbound, PermissionAIComplete}
}

// PluginCapabilities is every core capability the guide lists.
func PluginCapabilities() []string {
	return []string{
		"embedding",
		"tools",
		"agent",
		"ai.agent.provider",
	}
}

// PluginActivationEvents is every activation event pattern the guide lists.
func PluginActivationEvents() []string {
	return []string{
		"onCommand:<id>",
		"*",
	}
}

var (
	ErrEmptyID                = errors.New("plugin id cannot be empty")
	ErrEmptyName              = errors.New("plugin name cannot be empty")
	ErrEmptyVersion           = errors.New("plugin version cannot be empty")
	ErrInvalidID              = errors.New("plugin id must be reverse-DNS: two or more dot-separated segments of letters, digits, _ or - (e.g. org.example.sample)")
	ErrInvalidVersion         = errors.New("plugin version is not valid semver (e.g. 1.0.0)")
	ErrUnsupportedProtocol    = errors.New("unsupported plugin protocol")
	ErrInvalidLifecycle       = errors.New("invalid lifecycle")
	ErrEmptySettingTitle      = errors.New("setting title cannot be empty")
	ErrInvalidBackend         = errors.New("invalid backend")
	ErrInvalidFrontend        = errors.New("invalid frontend")
	ErrInvalidPermission      = errors.New("invalid permission")
	ErrInvalidCapability      = errors.New("invalid capability")
	ErrInvalidActivationEvent = errors.New("invalid activation event")
	ErrInvalidRuntime         = errors.New("invalid runtime")
	ErrEmptySettingKey        = errors.New("setting key cannot be empty")
	ErrInvalidSettingKey      = errors.New("invalid setting key")
	ErrDuplicateSettingKey    = errors.New("duplicate setting key")
	ErrInvalidSettingType     = errors.New("invalid setting type")
	ErrEnumMissingOptions     = errors.New("enum setting must define options")
	ErrRangeInvalidBounds     = errors.New("range min cannot exceed max")
	ErrInvalidSecret          = errors.New("invalid secret")
	ErrReservedSettingKey     = errors.New("the setting key is kept by the host")
	ErrInvalidAgentSetting    = errors.New("invalid agent setting")
	ErrInvalidModelsOf        = errors.New("only a string setting is offered the models of a program")
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
// absent means not asked yet. The window writes it, and no manifest takes the name.
const SettingAIConsent = "aiConsent"

// SettingTypeAgent is the type of the setting that holds the agent and the model
// answering the completions of a plugin. Its value is a CompletionChoice, and
// the user sets it: the setting declares no default.
const SettingTypeAgent = "agent"

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

// The types of a setting that holds a path on this machine.
const (
	SettingTypeFile   = "file"
	SettingTypeFolder = "folder"
)

var validSettingTypes = []string{"boolean", SettingTypeString, "number", SettingTypeEnum, "select", "range", SettingTypeAgent, SettingTypeFile, SettingTypeFolder}

// PluginLifecycle defines when the host evicts an idle plugin from memory.
type PluginLifecycle struct {
	IdleTimeoutSeconds    int
	KeepAliveWhileVisible bool
	MemoryLimitBytes      uint64
}

// PluginBackend describes the server-side executable of a plugin.
type PluginBackend struct {
	Type        string
	Entrypoint  string
	ServiceName string
}

// PluginFrontend describes the client-side bundle and styles of a plugin.
type PluginFrontend struct {
	Entrypoint string
	Styles     string
}

// PluginManifestSetting declares one configuration field managed by the core host.
type PluginManifestSetting struct {
	Key         string
	Type        string
	Title       string
	Description string
	Default     any
	Palette     bool
	Icon        string
	Options     PluginSettingOptions
	Min         float64
	Max         float64
	Step        float64
	// ModelsOf is the agent program whose models a string setting is offered, as the program names
	// itself. The models are those the plugins list for it.
	ModelsOf string
	// Replaces are the places in the settings file where the user stored this setting before the plugin took
	// it over, each written as the dot-joined keys leading to it, the newest first. The first of
	// them the file contains supplies the value for a user whose file has none for the plugin yet.
	Replaces []string
	// OptionsBy gives an enum setting its options from the value of another setting. The first option
	// of each list is the default under that value.
	OptionsBy *PluginSettingOptionsBy
	// When is the condition under which the setting is shown and handed to the plugin. A setting
	// whose condition fails keeps its stored value.
	When *PluginSettingCondition
	// Accept are the file extensions a setting of type file takes, each with its leading dot. Empty
	// takes any file.
	Accept []string
}

// PluginManifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type PluginManifest struct {
	ID               string
	Name             string
	Version          string
	Protocol         int
	Description      string
	Capabilities     []string
	Targets          []string
	ActivationEvents []string
	Permissions      []string
	Runtimes         []string
	Dependencies     []DependencyRequirement
	Lifecycle        PluginLifecycle
	Backend          PluginBackend
	Frontend         PluginFrontend
	Settings         []PluginManifestSetting
	// Secrets are the names of the secrets, such as an API key, the plugin asks the
	// host to store. The host passes them as Configure values, empty while a
	// secret has no key, and never puts one in the environment or on the command line.
	Secrets []PluginSecret
	// Services are the services the plugin serves to the windows, by their full names. A window that
	// receives a call for one forwards it to the plugin; a service the host serves itself is never
	// forwarded.
	Services []string
	// NoteTypes are the types of note the plugin gives meaning to, and how the index keeps the notes of
	// each. A note of a type no plugin declares is kept as an ordinary note is.
	NoteTypes []PluginNoteType
	// Commands are the commands the plugin's backend runs, offered by the palette and the command line.
	Commands []PluginCommand
	// Texts is what the plugin declares of the text it makes. A plugin that makes no text declares none.
	Texts *PluginTextMaker
	// Playback is the file types the plugin keeps a copy to play of. A plugin that keeps none declares none.
	Playback *PluginPlayback
	// ContentTypes are the types of file the plugin brings that the content type table does not know.
	ContentTypes []PluginContentType
	// Readers are the tabs of the plugin's frontend that open files, each by the id the frontend
	// registers it under.
	Readers []PluginReader
	// SettingsViews are the components of the plugin's frontend the settings page draws among the
	// settings of the plugin, each by the id the frontend registers it under.
	SettingsViews []PluginSettingsView
	// Frames are the https origins the plugin's pages frame, such as a video site's player. The
	// window's frame-src names them while the plugin is enabled.
	Frames []string
}

// PluginNoteType declares one type of note and how the index keeps the notes of it.
type PluginNoteType struct {
	// Type is the word the `type` key of a note's frontmatter holds.
	Type string
	// Chunks says whether the prose of the note is split into chunks and searched by meaning. When omitted,
	// it is.
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

var decodeManifest func([]byte) (PluginManifest, error)

// SetManifestDecoder registers the function that decodes a manifest from bytes.
func SetManifestDecoder(fn func([]byte) (PluginManifest, error)) {
	decodeManifest = fn
}

// UnmarshalJSON decodes a manifest using the registered decoder.
func (m *PluginManifest) UnmarshalJSON(data []byte) error {
	if decodeManifest == nil {
		return errors.New("manifest decoder not registered")
	}
	parsed, err := decodeManifest(data)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// ReadManifest decodes and validates a manifest from input.
func ReadManifest(r io.Reader) (PluginManifest, error) {
	if decodeManifest == nil {
		return PluginManifest{}, errors.New("manifest decoder not registered")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return PluginManifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	m, err := decodeManifest(data)
	if err != nil {
		return PluginManifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return PluginManifest{}, err
	}
	return m, nil
}

// GetProtocol returns the plugin protocol the manifest declares. An absent protocol is 1.
func (m PluginManifest) GetProtocol() int {
	if m.Protocol == 0 {
		return 1
	}
	return m.Protocol
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
	issues = append(issues, m.listBackendlessIssues()...)
	issues = append(issues, m.listPermissionIssues()...)
	issues = append(issues, m.listRuntimeIssues()...)
	issues = append(issues, m.listDependencyIssues()...)
	issues = append(issues, m.listSecretIssues()...)
	issues = append(issues, m.listServiceIssues()...)
	issues = append(issues, m.listNoteTypeIssues()...)
	issues = append(issues, m.listCommandIssues()...)
	issues = append(issues, m.listTextsIssues()...)
	issues = append(issues, m.listPlaybackIssues()...)
	issues = append(issues, m.listContentTypeIssues()...)
	issues = append(issues, m.listReaderIssues()...)
	issues = append(issues, m.listSettingsViewIssues()...)
	issues = append(issues, m.listFrameIssues()...)
	issues = append(issues, m.listAgentSettingIssues()...)
	issues = append(issues, m.listSettingLinkIssues()...)
	return append(issues, listManifestSettingIssues(m.Settings)...)
}

// listServiceIssues checks that each service is a full name of dot-separated words, and is listed once.
func (m PluginManifest) listServiceIssues() []error {
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
func (m PluginManifest) listNoteTypeIssues() []error {
	var issues []error
	observed := map[string]bool{}
	for _, declared := range m.NoteTypes {
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
func (t PluginNoteType) GetProfile() domain.NoteProfile {
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
		profile.IDPattern, _ = regexp.Compile("^(?:" + t.IDPattern + ")$")
	}
	return profile
}

// GetNoteType is the declaration of a type of note the plugin gives meaning to.
func (m PluginManifest) GetNoteType(noteType string) (PluginNoteType, bool) {
	for _, declared := range m.NoteTypes {
		if declared.Type == noteType {
			return declared, true
		}
	}
	return PluginNoteType{}, false
}

// listAgentSettingIssues checks the setting that chooses the agent: one at most,
// declared by a plugin that holds the ai:complete permission, with no default.
func (m PluginManifest) listAgentSettingIssues() []error {
	var issues []error
	count := 0
	for _, s := range m.Settings {
		if s.Type != SettingTypeAgent {
			continue
		}
		count++
		if s.Default != nil {
			issues = append(issues, fmt.Errorf("%w: %s declares a default", ErrInvalidAgentSetting, s.Key))
		}
	}
	if count > 1 {
		issues = append(issues, fmt.Errorf("%w: %d are declared, and a plugin has one at most", ErrInvalidAgentSetting, count))
	}
	if count > 0 && !m.HasPermission(PermissionAIComplete) {
		issues = append(issues, fmt.Errorf("%w: the plugin does not hold %s", ErrInvalidAgentSetting, PermissionAIComplete))
	}
	return issues
}

// GetAgentSettingKey returns the key of the setting that chooses the agent of the
// plugin's completions, and false when the manifest declares none.
func (m PluginManifest) GetAgentSettingKey() (string, bool) {
	for _, s := range m.Settings {
		if s.Type == SettingTypeAgent {
			return s.Key, true
		}
	}
	return "", false
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

// HasBackend reports whether the plugin declares a process the host starts.
func (m PluginManifest) HasBackend() bool {
	return strings.TrimSpace(m.Backend.Entrypoint) != ""
}

// HasFrontend reports whether the plugin declares a bundle the windows load.
func (m PluginManifest) HasFrontend() bool {
	return strings.TrimSpace(m.Frontend.Entrypoint) != ""
}

// listBackendlessIssues names each field of a plugin without a backend that only a backend serves:
// the host would start a process to answer it.
func (m PluginManifest) listBackendlessIssues() []error {
	if m.Backend != (PluginBackend{}) {
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
		{"runtimes", len(m.Runtimes) > 0},
		{"texts", m.Texts != nil},
		{"playback", m.Playback != nil},
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
func (m PluginManifest) ValidatePermissions() error {
	return errors.Join(m.listPermissionIssues()...)
}

func (m PluginManifest) listPermissionIssues() []error {
	var issues []error
	for _, p := range m.Permissions {
		if !slices.Contains(PluginPermissions(), NormalizePermission(p)) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidPermission, p))
		}
	}
	return issues
}

// ValidateCapabilities checks that every capability is one the validator recognizes.
func (m PluginManifest) ValidateCapabilities() error {
	var issues []error
	for _, c := range m.Capabilities {
		if !slices.Contains(PluginCapabilities(), c) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidCapability, c))
		}
	}
	return errors.Join(issues...)
}

// ValidateActivationEvents checks that every activation event matches a recognized pattern.
func (m PluginManifest) ValidateActivationEvents() error {
	var issues []error
	for _, e := range m.ActivationEvents {
		if !isRecognizedActivationEvent(e) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidActivationEvent, e))
		}
	}
	return errors.Join(issues...)
}

func isRecognizedActivationEvent(event string) bool {
	if event == "*" {
		return true
	}
	if strings.HasPrefix(event, "onCommand:") && len(event) > len("onCommand:") {
		return true
	}
	return false
}

func (m PluginManifest) listRuntimeIssues() []error {
	var issues []error
	observed := make(map[string]bool, len(m.Runtimes))
	for _, name := range m.Runtimes {
		switch {
		case name != RuntimeONNX:
			issues = append(issues, fmt.Errorf("%w: %q, expected %q", ErrInvalidRuntime, name, RuntimeONNX))
		case observed[name]:
			issues = append(issues, fmt.Errorf("%w: %s is declared twice", ErrInvalidRuntime, name))
		}
		observed[name] = true
	}
	return issues
}

func (m PluginManifest) listSecretIssues() []error {
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
		case slices.ContainsFunc(m.Settings, func(s PluginManifestSetting) bool { return s.Key == name }):
			issues = append(issues, fmt.Errorf("%w: %q is also a setting", ErrInvalidSecret, name))
		}
		observed[name] = true
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

func listManifestSettingIssue(s PluginManifestSetting, key string) []error {
	var issues []error
	if strings.TrimSpace(s.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEmptySettingTitle, key))
	}
	if !slices.Contains(validSettingTypes, s.Type) {
		issues = append(issues, fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key))
	}
	if (s.Type == SettingTypeEnum || s.Type == "select") && len(s.Options) == 0 && s.OptionsBy == nil {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEnumMissingOptions, key))
	}
	if s.Type == "range" && (s.Min != 0 || s.Max != 0) && s.Min > s.Max {
		issues = append(issues, fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key))
	}
	if s.ModelsOf != "" && s.Type != "string" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidModelsOf, key))
	}
	return append(issues, listAcceptIssues(s, key)...)
}

// listAcceptIssues checks that only a file setting accepts extensions, each with its leading dot.
func listAcceptIssues(s PluginManifestSetting, key string) []error {
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

// HasService reports whether this plugin serves the service to the windows.
func (m PluginManifest) HasService(service string) bool {
	if service == "" {
		return false
	}
	return (m.Backend.ServiceName != "" && m.Backend.ServiceName == service) || slices.Contains(m.Services, service)
}

// CanActivateOn reports whether the given event triggers plugin activation.
func (m PluginManifest) CanActivateOn(event string) bool {
	return slices.Contains(m.ActivationEvents, "*") || slices.Contains(m.ActivationEvents, event)
}
