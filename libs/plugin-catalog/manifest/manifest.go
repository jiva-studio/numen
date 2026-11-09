package manifest

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = "vault:read"

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = "vault:write"

	// PermissionVaultManage allows adding, renaming, removing and opening the vaults of the
	// installation.
	PermissionVaultManage = "vault:manage"

	// PermissionNetworkOutbound allows outbound network requests.
	PermissionNetworkOutbound = "network:outbound"

	// PermissionAIComplete allows asking the host for a model completion.
	PermissionAIComplete = "ai:complete"

	// PermissionIndexRead allows reading the search index of a vault: its note titles, the stored
	// vectors of its chunks, and the state of its embedding.
	PermissionIndexRead = "index:read"

	// PermissionSearchPolicy allows reading and setting the floor that a search by meaning applies.
	PermissionSearchPolicy = "search:policy"

	// PermissionWindowFocus allows focusing the window of the vault on a note or a file, and reading
	// the tabs it has open.
	PermissionWindowFocus = "window:focus"

	// PermissionNetworkAlias is the accepted short spelling of PermissionNetworkOutbound.
	PermissionNetworkAlias = "network"

	ProtocolVersion = 1
)

// PluginPermissions is every permission a manifest may declare, in its canonical spelling.
func PluginPermissions() []string {
	return []string{
		PermissionVaultRead, PermissionVaultWrite, PermissionVaultManage, PermissionNetworkOutbound,
		PermissionAIComplete, PermissionIndexRead, PermissionSearchPolicy, PermissionWindowFocus,
	}
}

const (
	// ActivationEventStartup is the event the host emits when it starts.
	ActivationEventStartup = "onStartup"

	// ActivationEventVectorsReady is the event the host emits when the vectors of a vault are ready.
	ActivationEventVectorsReady = "onVectorsReady"

	// ActivationEventFilePrefix starts the event the host emits when a file is opened or listed.
	ActivationEventFilePrefix = "onFile:"
)

// PluginActivationEvents is every activation event pattern a manifest may declare.
func PluginActivationEvents() []string {
	return []string{
		ActivationEventStartup,
		ActivationEventVectorsReady,
		ActivationEventFilePrefix + "<ext>",
	}
}

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

// IsValidID reports whether the text is the shape of a plugin id: two or more dot-separated
// segments of letters, digits, _ or -. The ids `app` and `v`, which the first level of the
// settings file keeps, are no plugin ids.
func IsValidID(id string) bool { return manifestIDPattern.MatchString(id) }

// SettingEnabled is the field of the object of a plugin in the settings file that holds whether the
// host runs the plugin. No setting of a manifest takes the name.
const SettingEnabled = "enabled"

func listReservedSettingKeys() []string { return []string{SettingEnabled} }

var (
	manifestIDPattern      = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	manifestServicePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
	manifestSecretPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	manifestVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))(\.((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)))*)?(\+([0-9A-Za-z-]+)(\.[0-9A-Za-z-]+)*)?$`)
)

// SettingTypeString is the type of a setting that holds a text string.
const SettingTypeString = "string"

// SettingGenerateToken makes the host store 32 random bytes in hex in a setting that holds no value.
const SettingGenerateToken = "token"

// SettingTypeEnum is the type of a setting that holds one of its options.
const SettingTypeEnum = "enum"

// SettingTypeFile is the type of a setting that holds a file path on this machine.
const SettingTypeFile = "file"

// settingTypeNumber is the type of a setting that stores a number.
const settingTypeNumber = "number"

func listSettingTypes() []string {
	return []string{"boolean", SettingTypeString, settingTypeNumber, SettingTypeEnum, "range", SettingTypeFile}
}

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
	Min         *float64
	Max         *float64
	Step        *float64
	OptionsBy   *OptionsBy
	When        *Condition
	Accept      []string
	Multiple    bool
	Preview     bool
	OptionsFrom string
	Generate    string
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
	Frontend         *Frontend
	Settings         []Setting
	Secrets          []SecretDeclaration
	Services         []string
	SettingsViews    []SettingsView
	Contributes      Contributes
	Warnings         []string
}

// NoteType declares one type of note and how the index keeps the notes of it.
type NoteType struct {
	Type            string
	Chunks          *bool
	Headings        *[]int
	IDHeadingLevels []int
	IDPattern       string
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
		issues = append(issues, fmt.Errorf("%w: %d, this host supports up to %d", ErrUnsupportedProtocol, m.Protocol, ProtocolVersion))
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

func (m Manifest) listBackendlessIssues() []error {
	if m.Backend != (Backend{}) {
		return nil
	}
	fields := []struct {
		name       string
		isDeclared bool
	}{
		{"activationEvents", len(m.ActivationEvents) > 0},
		{"services", len(m.Services) > 0},
		{"secrets", len(m.Secrets) > 0},
		{"runtime dependencies", slices.ContainsFunc(m.Dependencies, func(d DependencyRequirement) bool { return d.Kind == DependencyKindRuntime })},
		{"contributes.triggers", len(m.Contributes.Triggers) > 0},
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

func (m Manifest) listCapabilityIssues() []error {
	var issues []error
	for _, c := range m.Capabilities {
		if !slices.Contains(PluginCapabilities(), c) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidCapability, c))
		}
	}
	return issues
}

func (m Manifest) listActivationEventIssues() []error {
	var issues []error
	for _, e := range m.ActivationEvents {
		if !isValidActivationEvent(e) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidActivationEvent, e))
		}
	}
	return issues
}

func isValidActivationEvent(event string) bool {
	if event == ActivationEventStartup || event == ActivationEventVectorsReady {
		return true
	}
	ext, ok := strings.CutPrefix(event, ActivationEventFilePrefix)
	return ok && ext != "" && strings.TrimSpace(ext) == ext && !strings.Contains(ext, "*")
}

// maxSecretGroupLength is the longest group name of a secret declaration, in characters.
const maxSecretGroupLength = 64

func (m Manifest) listSecretIssues() []error {
	var issues []error
	for _, secret := range m.Secrets {
		name := secret.Name
		issues = append(issues, m.listConditionIssues("secret "+name, secret.When)...)
		issues = append(issues, listSecretGroupIssues(secret)...)
		switch {
		case !manifestSecretPattern.MatchString(name):
			issues = append(issues, fmt.Errorf("%w: %q is not letters, digits, _ or -", ErrInvalidSecret, name))
		case slices.Contains(listReservedSettingKeys(), name):
			issues = append(issues, fmt.Errorf("%w: secret %q", ErrReservedSettingKey, name))
		case slices.ContainsFunc(m.Settings, func(s Setting) bool { return s.Key == name }):
			issues = append(issues, fmt.Errorf("%w: %q is also a setting", ErrInvalidSecret, name))
		}
	}
	return append(issues, listRepeatedSecretIssues(m.Secrets)...)
}

func listSecretGroupIssues(secret SecretDeclaration) []error {
	group := secret.Group
	if group == "" {
		return nil
	}
	switch {
	case strings.TrimSpace(group) != group:
		return []error{fmt.Errorf("%w: group %q of %q has leading or trailing space", ErrInvalidSecret, group, secret.Name)}
	case utf8.RuneCountInString(group) > maxSecretGroupLength:
		return []error{fmt.Errorf("%w: group of %q is longer than %d characters", ErrInvalidSecret, secret.Name, maxSecretGroupLength)}
	}
	return nil
}

func listRepeatedSecretIssues(secrets []SecretDeclaration) []error {
	var issues []error
	byName := make(map[string][]SecretDeclaration, len(secrets))
	for _, secret := range secrets {
		byName[secret.Name] = append(byName[secret.Name], secret)
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		declared := byName[name]
		if len(declared) < 2 {
			continue
		}
		if !isDisjoint(declared) {
			issues = append(issues, fmt.Errorf("%w: %q is declared twice without conditions on one setting that exclude each other", ErrInvalidSecret, name))
			continue
		}
		groups := make(map[string]bool, len(declared))
		for _, secret := range declared {
			if groups[secret.Group] {
				issues = append(issues, fmt.Errorf("%w: %q is declared twice for the group %q", ErrInvalidSecret, name, secret.Group))
			}
			groups[secret.Group] = true
		}
	}
	return issues
}

func isDisjoint(declared []SecretDeclaration) bool {
	values := make(map[string]bool)
	for _, secret := range declared {
		if secret.When == nil || secret.When.Setting != declared[0].When.Setting {
			return false
		}
		for _, is := range secret.When.Is {
			if values[is] {
				return false
			}
		}
		for _, is := range secret.When.Is {
			values[is] = true
		}
	}
	return true
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
		if slices.Contains(listReservedSettingKeys(), key) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrReservedSettingKey, key))
			continue
		}
		if seenKeys[key] {
			issues = append(issues, fmt.Errorf("%w: %s", ErrDuplicateSettingKey, key))
			continue
		}
		seenKeys[key] = true
		if strings.Contains(key, ".") {
			issues = append(issues, fmt.Errorf("%w: %s holds a dot, and a key is one segment of a path", ErrInvalidSettingKey, key))
		}
		issues = append(issues, listManifestSettingIssue(s, key)...)
	}
	return issues
}

func listManifestSettingIssue(s Setting, key string) []error {
	var issues []error
	if strings.TrimSpace(s.Title) == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEmptySettingTitle, key))
	}
	if !slices.Contains(listSettingTypes(), s.Type) {
		issues = append(issues, fmt.Errorf("%w: %q for %s", ErrInvalidSettingType, s.Type, key))
	}
	if s.Type == SettingTypeEnum && len(s.Options) == 0 && s.OptionsBy == nil && s.OptionsFrom == "" {
		issues = append(issues, fmt.Errorf("%w: %s", ErrEnumMissingOptions, key))
	}
	if s.Type == "range" && s.Min != nil && s.Max != nil && *s.Min > *s.Max {
		issues = append(issues, fmt.Errorf("%w: %s", ErrRangeInvalidBounds, key))
	}
	issues = append(issues, listGenerateSettingIssues(s, key)...)
	return append(issues, listFileSettingIssues(s, key)...)
}

func listGenerateSettingIssues(s Setting, key string) []error {
	if s.Generate == "" {
		return nil
	}
	var issues []error
	if s.Generate != SettingGenerateToken {
		issues = append(issues, fmt.Errorf("%w: %s generates %q, and the host generates only %q", ErrInvalidSetting, key, s.Generate, SettingGenerateToken))
	}
	if s.Type != SettingTypeString {
		issues = append(issues, fmt.Errorf("%w: %s generates a value and is no string setting", ErrInvalidSetting, key))
	}
	if s.Default != nil {
		issues = append(issues, fmt.Errorf("%w: %s generates a value and declares a default", ErrInvalidSetting, key))
	}
	return issues
}

func listFileSettingIssues(s Setting, key string) []error {
	var issues []error
	if len(s.Accept) > 0 && s.Type != SettingTypeFile {
		issues = append(issues, fmt.Errorf("%w: %s accepts extensions and holds no file", ErrInvalidSetting, key))
	}
	if s.Multiple && s.Type != SettingTypeFile {
		issues = append(issues, fmt.Errorf("%w: %s holds several paths and is no file setting", ErrInvalidSetting, key))
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

// IsActivatedBy reports whether the plugin declares an activation event that matches event.
func (m Manifest) IsActivatedBy(event string) bool {
	for _, declared := range m.ActivationEvents {
		if isMatchingActivationEvent(declared, event) {
			return true
		}
	}
	return false
}

func isMatchingActivationEvent(declared, event string) bool {
	switch declared {
	case ActivationEventStartup:
		return event == ActivationEventStartup
	case ActivationEventVectorsReady:
		return event == ActivationEventVectorsReady ||
			(strings.HasPrefix(event, ActivationEventVectorsReady+":") && len(event) > len(ActivationEventVectorsReady)+1)
	default:
		return isMatchingFileEvent(declared, event)
	}
}

func isMatchingFileEvent(declared, event string) bool {
	ext, ok := strings.CutPrefix(declared, ActivationEventFilePrefix)
	if !ok || ext == "" || strings.Contains(ext, "*") {
		return false
	}
	target, ok := strings.CutPrefix(event, ActivationEventFilePrefix)
	if !ok || target == "" {
		return false
	}
	if dot := strings.LastIndexByte(target, '.'); dot >= 0 {
		target = target[dot+1:]
	}
	return target != "" && strings.EqualFold(ext, target)
}
