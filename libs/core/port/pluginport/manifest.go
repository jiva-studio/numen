package pluginport

import "github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

// The manifest of a plugin, as the plugin catalog library reads and validates it.
type (
	DependencyKind         = manifest.DependencyKind
	DependencyRequirement  = manifest.DependencyRequirement
	DiscoveredPlugin       = manifest.Discovered
	PluginBackend          = manifest.Backend
	PluginCommand          = manifest.Command
	PluginCommandArgument  = manifest.CommandArgument
	PluginCommandWhen      = manifest.CommandWhen
	PluginContentType      = manifest.DeclaredContentType
	PluginFrontend         = manifest.Frontend
	PluginLifecycle        = manifest.Lifecycle
	PluginManifest         = manifest.Manifest
	PluginManifestSetting  = manifest.Setting
	PluginNoteType         = manifest.NoteType
	PluginPlayback         = manifest.Playback
	PluginReader           = manifest.Reader
	PluginSecret           = manifest.Secret
	PluginSettingsView     = manifest.SettingsView
	PluginSettingCondition = manifest.Condition
	PluginSettingOption    = manifest.SettingOption
	PluginSettingOptions   = manifest.SettingOptions
	PluginSettingOptionsBy = manifest.OptionsBy
	PluginTextMaker        = manifest.TextMaker
)

const (
	PermissionVaultRead       = manifest.PermissionVaultRead
	PermissionVaultWrite      = manifest.PermissionVaultWrite
	PermissionNetworkOutbound = manifest.PermissionNetworkOutbound
	PermissionAIComplete      = manifest.PermissionAIComplete
	PermissionNetworkAlias    = manifest.PermissionNetworkAlias
	RuntimeONNX               = manifest.RuntimeONNX
	PluginProtocolVersion     = manifest.ProtocolVersion
	SettingStoredSecrets      = manifest.SettingStoredSecrets
	SettingAIConsent          = manifest.SettingAIConsent
	SettingTypeAgent          = manifest.SettingTypeAgent
	SettingTypeString         = manifest.SettingTypeString
	SettingTypeEnum           = manifest.SettingTypeEnum
	SettingTypeFile           = manifest.SettingTypeFile
	SettingTypeFolder         = manifest.SettingTypeFolder
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
	ErrInvalidAgentSetting    = manifest.ErrInvalidAgentSetting
	ErrInvalidBackend         = manifest.ErrInvalidBackend
	ErrInvalidCapability      = manifest.ErrInvalidCapability
	ErrInvalidCommand         = manifest.ErrInvalidCommand
	ErrInvalidCommandArgument = manifest.ErrInvalidCommandArgument
	ErrInvalidDependency      = manifest.ErrInvalidDependency
	ErrInvalidFrame           = manifest.ErrInvalidFrame
	ErrInvalidFrontend        = manifest.ErrInvalidFrontend
	ErrInvalidID              = manifest.ErrInvalidID
	ErrInvalidLifecycle       = manifest.ErrInvalidLifecycle
	ErrInvalidModelsOf        = manifest.ErrInvalidModelsOf
	ErrInvalidNoteType        = manifest.ErrInvalidNoteType
	ErrInvalidPermission      = manifest.ErrInvalidPermission
	ErrInvalidPlayback        = manifest.ErrInvalidPlayback
	ErrInvalidReader          = manifest.ErrInvalidReader
	ErrInvalidRuntime         = manifest.ErrInvalidRuntime
	ErrInvalidSecret          = manifest.ErrInvalidSecret
	ErrInvalidService         = manifest.ErrInvalidService
	ErrInvalidSetting         = manifest.ErrInvalidSetting
	ErrInvalidSettingKey      = manifest.ErrInvalidSettingKey
	ErrInvalidSettingsView    = manifest.ErrInvalidSettingsView
	ErrInvalidSettingType     = manifest.ErrInvalidSettingType
	ErrInvalidTexts           = manifest.ErrInvalidTexts
	ErrInvalidVersion         = manifest.ErrInvalidVersion
	ErrNoBackend              = manifest.ErrNoBackend
	ErrRangeInvalidBounds     = manifest.ErrRangeInvalidBounds
	ErrReservedSettingKey     = manifest.ErrReservedSettingKey
	ErrUnsupportedProtocol    = manifest.ErrUnsupportedProtocol
	FormatSettingValue        = manifest.FormatSettingValue
	GetDependencyKinds        = manifest.GetDependencyKinds
	NormalizePermission       = manifest.NormalizePermission
	PluginActivationEvents    = manifest.PluginActivationEvents
	PluginCapabilities        = manifest.PluginCapabilities
	PluginPermissions         = manifest.PluginPermissions
)
