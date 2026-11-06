package pluginport

import (
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin-catalog/when"
)

// The manifest of a plugin, as the plugin catalog library reads and validates it.
type (
	DependencyKind         = manifest.DependencyKind
	DependencyRequirement  = manifest.DependencyRequirement
	DiscoveredPlugin       = manifest.Discovered
	PluginBackend          = manifest.Backend
	PluginCommand          = manifest.Command
	PluginCommandArgument  = manifest.CommandArgument
	PluginCondition        = when.Condition
	PluginContributes      = manifest.Contributes
	PluginFrontend         = manifest.Frontend
	PluginManifest         = manifest.Manifest
	PluginManifestSetting  = manifest.Setting
	PluginNoteType         = manifest.NoteType
	PluginReader           = manifest.Reader
	PluginSecret           = manifest.Secret
	PluginSettingsView     = manifest.SettingsView
	PluginSettingCondition = manifest.Condition
	PluginSettingOption    = manifest.SettingOption
	PluginSettingOptions   = manifest.SettingOptions
	PluginSettingOptionsBy = manifest.OptionsBy
)

// ConfigPerformanceProfile is the entry of the Configure values that holds the performance profile; plugin-sdk-go names it performance.ConfigKey.
const ConfigPerformanceProfile = "host.performanceProfile"

const (
	PermissionVaultRead       = manifest.PermissionVaultRead
	PermissionVaultWrite      = manifest.PermissionVaultWrite
	PermissionNetworkOutbound = manifest.PermissionNetworkOutbound
	PermissionAIComplete      = manifest.PermissionAIComplete
	PermissionIndexRead       = manifest.PermissionIndexRead
	PermissionSearchPolicy    = manifest.PermissionSearchPolicy
	PermissionNetworkAlias    = manifest.PermissionNetworkAlias
	RuntimeONNX               = "onnxruntime"
	ActivationEventAll        = manifest.ActivationEventAll
	ConditionTrue             = when.True
	ConditionFalse            = when.False
	ConditionUnknown          = when.Unknown
	PluginProtocolVersion     = manifest.ProtocolVersion
	SettingStoredSecrets      = manifest.SettingStoredSecrets
	SettingTypeString         = manifest.SettingTypeString
	SettingTypeEnum           = manifest.SettingTypeEnum
	SettingTypeFile           = manifest.SettingTypeFile
	CommandArgumentString     = manifest.CommandArgumentString
	CommandArgumentNumber     = manifest.CommandArgumentNumber
	CommandArgumentBoolean    = manifest.CommandArgumentBoolean
	CommandArgumentVersion    = manifest.CommandArgumentVersion
	DependencyKindRuntime     = manifest.DependencyKindRuntime
	DependencyKindModel       = manifest.DependencyKindModel
	DependencyKindBinary      = manifest.DependencyKindBinary
	DependencyKindDataset     = manifest.DependencyKindDataset
)

var (
	ErrDuplicateSettingKey    = manifest.ErrDuplicateSettingKey
	ErrEmptyID                = manifest.ErrEmptyID
	ErrEmptyName              = manifest.ErrEmptyName
	ErrEmptySettingKey        = manifest.ErrEmptySettingKey
	ErrEmptySettingTitle      = manifest.ErrEmptySettingTitle
	ErrEmptyVersion           = manifest.ErrEmptyVersion
	ErrEnumMissingOptions     = manifest.ErrEnumMissingOptions
	ErrInvalidActivationEvent = manifest.ErrInvalidActivationEvent
	ErrInvalidBackend         = manifest.ErrInvalidBackend
	ErrInvalidCapability      = manifest.ErrInvalidCapability
	ErrInvalidCommand         = manifest.ErrInvalidCommand
	ErrInvalidCommandArgument = manifest.ErrInvalidCommandArgument
	ErrInvalidDependency      = manifest.ErrInvalidDependency
	ErrInvalidFrame           = manifest.ErrInvalidFrame
	ErrInvalidID              = manifest.ErrInvalidID
	ErrInvalidNoteType        = manifest.ErrInvalidNoteType
	ErrInvalidPermission      = manifest.ErrInvalidPermission
	ErrInvalidReader          = manifest.ErrInvalidReader
	ErrInvalidSecret          = manifest.ErrInvalidSecret
	ErrInvalidService         = manifest.ErrInvalidService
	ErrInvalidSetting         = manifest.ErrInvalidSetting
	ErrInvalidSettingKey      = manifest.ErrInvalidSettingKey
	ErrInvalidSettingsView    = manifest.ErrInvalidSettingsView
	ErrInvalidSettingType     = manifest.ErrInvalidSettingType
	ErrInvalidVersion         = manifest.ErrInvalidVersion
	ErrNoBackend              = manifest.ErrNoBackend
	ErrRangeInvalidBounds     = manifest.ErrRangeInvalidBounds
	ErrReservedSettingKey     = manifest.ErrReservedSettingKey
	ErrUnsupportedProtocol    = manifest.ErrUnsupportedProtocol
)

func FormatSettingValue(value any) string { return manifest.FormatSettingValue(value) }

func GetDependencyKinds() []manifest.DependencyKind { return manifest.GetDependencyKinds() }

func NormalizePermission(permission string) string { return manifest.NormalizePermission(permission) }

func PluginCapabilities() []string { return manifest.PluginCapabilities() }

func PluginPermissions() []string { return manifest.PluginPermissions() }

// ParseCondition reads the when condition of a command or a slot entry; see package when.
func ParseCondition(text string) (PluginCondition, error) {
	return when.Parse(text)
}

// IsSharedCommand reports whether the full id is a shared command of libs/protocol/commands.json.
func IsSharedCommand(id string) bool { return manifest.IsSharedCommand(id) }

// The events of libs/protocol/events.json a trigger runs on.
const (
	EventFileAdded   = manifest.EventFileAdded
	EventFileChanged = manifest.EventFileChanged
)
