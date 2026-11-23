// Package manifest is the model of a plugin's manifest.json: the fields a plugin declares, the
// rules between them, what a plugin requires of the application, and the order of versions.
//
// The package reads no file and no JSON; manifestjson decodes the file into this model.
package manifest

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = "vault:read"

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = "vault:write"

	// PermissionVaultManage allows adding, renaming, removing and opening the vaults of the
	// installation.
	PermissionVaultManage = "vault:manage"

	// PermissionLinkOpen allows a plugin to open external links in the system browser. It does not
	// restrict network requests: a plugin is trusted code, and permissions are consent and API hygiene.
	PermissionLinkOpen = "link:open"

	// PermissionIndexRead allows reading the search index of a vault: its note titles, the stored
	// vectors of its chunks, and the state of its embedding.
	PermissionIndexRead = "index:read"

	// PermissionSearchPolicy allows reading and setting the floor that a search by meaning applies.
	PermissionSearchPolicy = "search:policy"

	// PermissionWindowFocus allows focusing the window of the vault on a note or a file, and reading
	// the tabs it has open.
	PermissionWindowFocus = "window:focus"

	// PermissionPluginsRead allows reading which plugins run, since when, how long each has been idle,
	// when the host stops it for idleness, and its worst health item.
	PermissionPluginsRead = "plugins:read"

	// PermissionLogsRead allows reading and watching the structured log entries of the application,
	// the database, the windows and the plugins.
	PermissionLogsRead = "logs:read"

	// PermissionIdentity allows asking the host to sign with the key of this device, under the id of the plugin,
	// and reading what is public of the identity of this device. The plugin never holds a secret key.
	PermissionIdentity = "identity"

	// PermissionIdentityLink allows carrying the link of a new device and the revoke of a device between the core and the
	// sync server: the relay channel, the sealed messages of a link, the chain to publish and the chain to adopt. The
	// plugin never sees the QR text or the confirmation code.
	PermissionIdentityLink = "identity:link"

	// PermissionConnectionsUse is the start of the permission connections:use:<operation>, which allows
	// calling the operation through every connection that serves it, and listing those connections and the
	// values of their row fields. No setting of rows of the plugin lists the operation.
	PermissionConnectionsUse = "connections:use:"
)

// ProtocolVersion is the plugin protocol version of this host. A test of manifestjson keeps it equal to
// the version of the protocol library.
const ProtocolVersion = 1

// ListPluginPermissions returns every permission a manifest may declare, in its canonical spelling, besides the
// permissions connections:use:<operation>.
func ListPluginPermissions() []string {
	return []string{
		PermissionVaultRead, PermissionVaultWrite, PermissionVaultManage, PermissionLinkOpen,
		PermissionIndexRead, PermissionSearchPolicy, PermissionWindowFocus, PermissionPluginsRead,
		PermissionLogsRead, PermissionIdentity, PermissionIdentityLink,
	}
}

// FormatConnectionsUse returns the permission to call the operation through every connection that serves it.
func FormatConnectionsUse(operation string) string {
	return PermissionConnectionsUse + operation
}

// ParseConnectionsUse returns the operation a permission connections:use:<operation> names, and false for
// any other permission.
func ParseConnectionsUse(permission string) (string, bool) {
	operation, isFound := strings.CutPrefix(permission, PermissionConnectionsUse)
	return operation, isFound && connectionOperationPattern.MatchString(operation)
}

// ListConnectionsUse returns the operations the permissions connections:use:<operation> of the manifest
// name, in the order of the permissions.
func (m Manifest) ListConnectionsUse() []string {
	var operations []string
	for _, p := range m.Permissions {
		if operation, isGrant := ParseConnectionsUse(p); isGrant {
			operations = append(operations, operation)
		}
	}
	return operations
}

// ActivationEventIndexChanged is the event the host emits when indexing changed the index of a vault.
const ActivationEventIndexChanged = "onIndexChanged"

// ActivationEventFilesChanged is the event the host emits when the index wrote changes into the file change
// feed of a vault.
const ActivationEventFilesChanged = "onFilesChanged"

// ListPluginActivationEvents returns every activation event pattern a manifest may declare.
func ListPluginActivationEvents() []string {
	return []string{ActivationEventIndexChanged, ActivationEventFilesChanged}
}

// ListPluginCapabilities returns every capability a manifest may declare: the capabilities the host resolves
// a plugin by.
func ListPluginCapabilities() []string {
	return []string{
		"search.source",
		"tools",
	}
}

var (
	ErrEmptyID      = errors.New("plugin id cannot be empty")
	ErrEmptyName    = errors.New("plugin name cannot be empty")
	ErrEmptyVersion = errors.New("plugin version cannot be empty")
	ErrInvalidID    = errors.New(
		"plugin id must be reverse-DNS: two or more dot-separated segments of letters, digits, _ or - (e.g. org.example.sample)",
	)
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
	ErrReservedSettingKey     = errors.New("the setting key is kept by the host")
	ErrInvalidSetting         = errors.New("invalid setting")
	ErrLongSettingDescription = errors.New("setting description is longer than one line")
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

// SettingDiscover is the field of the object of a plugin in the settings file that holds, for each setting
// of type connections that declares discover, whether it uses every connection that serves its
// operations while it holds no row. No setting of a manifest takes the name.
const SettingDiscover = "discover"

func listReservedSettingKeys() []string { return []string{SettingEnabled, SettingDiscover} }

var (
	manifestIDPattern       = regexp.MustCompile(`^[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`)
	manifestServicePattern  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z][A-Za-z0-9_]*)+$`)
	manifestNoteTypePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	manifestVersionPattern  = regexp.MustCompile(
		`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))(\.((0|[1-9][0-9]*)|([0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)))*)?(\+([0-9A-Za-z-]+)(\.[0-9A-Za-z-]+)*)?$`,
	)
)

// SettingTypeString is the type of a setting that holds a text string.
const SettingTypeString = "string"

// SettingGenerateToken makes the host store 32 random bytes in hex in a setting that holds no value.
const SettingGenerateToken = "token"

// SettingTypeEnum is the type of a setting that holds one of its options.
const SettingTypeEnum = "enum"

// SettingTypeFile is the type of a setting that holds a file path on this machine.
const SettingTypeFile = "file"

// SettingTypeConnections is the type of a setting that holds rows of connections: each names a
// connection and holds the values of the row fields of its type.
const SettingTypeConnections = "connections"

// SettingTypeList is the type of a setting that holds a list of distinct, non-empty strings.
const SettingTypeList = "list"

// SettingTypeSlots is the type of the setting that keeps the order and the visibility of the entries of
// the slots a plugin draws: an object by slot name, each a list of { id, hidden }.
const SettingTypeSlots = "slots"

const settingTypeNumber = "number"

// settingTypeBoolean is the type of a setting that stores true or false.
const settingTypeBoolean = CommandArgumentBoolean

func listSettingTypes() []string {
	return []string{settingTypeBoolean, SettingTypeString, settingTypeNumber, SettingTypeEnum, "range", SettingTypeFile, SettingTypeConnections, SettingTypeList, SettingTypeSlots}
}

// Backend describes the server-side executable of a plugin. A zero IdleStopSeconds takes the idle limit
// of the host. An empty Lifecycle is LifecycleOnDemand.
type Backend struct {
	Entrypoint      string
	IdleStopSeconds int
	Lifecycle       string
}

const (
	// LifecycleOnDemand starts the process when something uses the plugin and stops it after the idle limit.
	LifecycleOnDemand = "onDemand"
	// LifecyclePersistent keeps the process running while the plugin is enabled.
	LifecyclePersistent = "persistent"
)

// IsPersistent reports whether the host keeps the process running.
func (b Backend) IsPersistent() bool {
	return b.Lifecycle == LifecyclePersistent
}

func (b Backend) listLifecycleIssues() []error {
	switch {
	case b.Lifecycle != "" && b.Lifecycle != LifecycleOnDemand && b.Lifecycle != LifecyclePersistent:
		return []error{fmt.Errorf("%w: unknown lifecycle %q", ErrInvalidBackend, b.Lifecycle)}
	case b.IsPersistent() && b.IdleStopSeconds != 0:
		return []error{fmt.Errorf("%w: a persistent backend has no idle limit", ErrInvalidBackend)}
	}
	return nil
}

const (
	// MinIdleStopSeconds is the shortest idle limit a manifest may set.
	MinIdleStopSeconds = 30
	// MaxIdleStopSeconds is the longest idle limit a manifest may set.
	MaxIdleStopSeconds = 3600
)

// ErrInvalidIdleStop is an idle limit outside MinIdleStopSeconds and MaxIdleStopSeconds.
var ErrInvalidIdleStop = errors.New("invalid backend idle limit")

// GetIdleStopTime returns how long the process may run without a lease before the host stops it, and
// zero where the manifest sets no limit.
func (b Backend) GetIdleStopTime() time.Duration {
	return time.Duration(b.IdleStopSeconds) * time.Second
}

func (b Backend) listIdleStopIssues() []error {
	if b.IdleStopSeconds == 0 || (b.IdleStopSeconds >= MinIdleStopSeconds && b.IdleStopSeconds <= MaxIdleStopSeconds) {
		return nil
	}
	return []error{fmt.Errorf("%w: idleStopSeconds %d is outside %d to %d", ErrInvalidIdleStop, b.IdleStopSeconds, MinIdleStopSeconds, MaxIdleStopSeconds)}
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
	Details     string
	Group       string
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
	Operations  []string
	Discover    bool
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
	Services         []string
	SettingsViews    []SettingsView
	SettingGroups    []SettingGroup
	Contributes      Contributes
	IsOffByDefault   bool
	Warnings         []string
}

// Validate checks the manifest and returns every violation joined into one error.
func (m Manifest) Validate() error {
	return errors.Join(m.ListIssues()...)
}

// ListIssues returns every rule the manifest breaks, one error each.
func (m Manifest) ListIssues() []error {
	var issues []error
	issues = append(issues, m.listIdentityIssues()...)
	if !IsProtocolCompatible(m.Protocol, ProtocolVersion) {
		issues = append(issues, fmt.Errorf("%w: %d, this host uses %d", ErrUnsupportedProtocol, m.GetProtocol(), ProtocolVersion))
	}
	issues = append(issues, m.listCapabilityIssues()...)
	issues = append(issues, m.listActivationEventIssues()...)
	issues = append(issues, m.listEntrypointIssues()...)
	issues = append(issues, m.listBackendlessIssues()...)
	issues = append(issues, m.Backend.listIdleStopIssues()...)
	issues = append(issues, m.Backend.listLifecycleIssues()...)
	issues = append(issues, m.listPermissionIssues()...)
	issues = append(issues, m.listDependencyIssues()...)
	issues = append(issues, m.listServiceIssues()...)
	issues = append(issues, m.listNoteTypeIssues()...)
	issues = append(issues, m.listCommandIssues()...)
	issues = append(issues, m.listSettingsViewIssues()...)
	issues = append(issues, m.listSettingGroupIssues()...)
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
		{"runtime dependencies", slices.ContainsFunc(m.Dependencies, func(d DependencyRequirement) bool { return d.Kind == DependencyKindRuntime })},
		{"contributes.triggers", len(m.Contributes.Triggers) > 0},
		{"contributes.connectionTypes", len(m.Contributes.ConnectionTypes) > 0},
		{"contributes.searchFunctions", len(m.Contributes.SearchFunctions.Functions) > 0},
	}
	var issues []error
	for _, field := range fields {
		if field.isDeclared {
			issues = append(issues, fmt.Errorf("%w: %s", ErrNoBackend, field.name))
		}
	}
	return issues
}

// ValidatePermissions verifies that all declared permissions are recognized capabilities.
func (m Manifest) ValidatePermissions() error {
	return errors.Join(m.listPermissionIssues()...)
}

func (m Manifest) listPermissionIssues() []error {
	var issues []error
	for _, p := range m.Permissions {
		_, isGrant := ParseConnectionsUse(p)
		if !isGrant && !slices.Contains(ListPluginPermissions(), p) {
			issues = append(issues, fmt.Errorf("%w: %s", ErrInvalidPermission, p))
		}
	}
	for _, operation := range m.ListConnectionsUse() {
		for _, s := range m.Settings {
			if s.Type == SettingTypeConnections && slices.Contains(s.Operations, operation) {
				issues = append(issues, fmt.Errorf("%w: %s, and the setting %s holds rows for the operation", ErrInvalidPermission, FormatConnectionsUse(operation), s.Key))
			}
		}
	}
	return issues
}

func (m Manifest) listCapabilityIssues() []error {
	var issues []error
	for _, c := range m.Capabilities {
		if !slices.Contains(ListPluginCapabilities(), c) {
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
	if slices.Contains(ListPluginActivationEvents(), event) {
		return true
	}
	_, isMasked := getFilesChangedMask(event)
	return isMasked
}

func getFilesChangedMask(event string) (domain.FileType, bool) {
	text, hasPrefix := strings.CutPrefix(event, ActivationEventFilesChanged+":")
	if !hasPrefix {
		return domain.FileType{}, false
	}
	if note, ok := strings.CutPrefix(text, "note:"); ok {
		return domain.FileType{Note: domain.NoteType(note)}, manifestNoteTypePattern.MatchString(note)
	}
	mask, err := domain.ParseContentType(text)
	return domain.FileType{Content: mask}, err == nil
}

// IsTakingFileTypes reports whether the plugin takes a file of one of the types: it declares onFilesChanged
// without a mask, or with a mask one of the types matches, or declares no onFilesChanged event and so
// filters nothing.
func (m Manifest) IsTakingFileTypes(types []domain.FileType) bool {
	isMasked := false
	for _, declared := range m.ActivationEvents {
		if declared == ActivationEventFilesChanged {
			return true
		}
		mask, hasMask := getFilesChangedMask(declared)
		if hasMask && slices.ContainsFunc(types, func(file domain.FileType) bool {
			if mask.Note != "" {
				return mask.Note == file.Note
			}
			return mask.Content.IsMatch(file.Content)
		}) {
			return true
		}
		isMasked = isMasked || hasMask
	}
	return !isMasked
}

// HasPermission reports whether this plugin requests the given permission.
func (m Manifest) HasPermission(permission string) bool {
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
	if !isValidActivationEvent(declared) {
		return false
	}
	if _, isMasked := getFilesChangedMask(declared); isMasked {
		declared = ActivationEventFilesChanged
	}
	vaultID, isScoped := strings.CutPrefix(event, declared+":")
	return event == declared || (isScoped && vaultID != "")
}
