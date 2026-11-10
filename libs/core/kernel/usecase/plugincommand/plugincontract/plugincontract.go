// Package plugincontract re-exports the plugin port contract.
package plugincontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

type (
	DependencyCatalog      = pluginport.DependencyCatalog
	DependencyKind         = pluginport.DependencyKind
	DependencyProvisioner  = pluginport.DependencyProvisioner
	DependencyRequirement  = pluginport.DependencyRequirement
	DiscoveredPlugin       = pluginport.DiscoveredPlugin
	KeyStore               = pluginport.KeyStore
	Lease                  = pluginport.Lease
	PluginActivator        = pluginport.PluginActivator
	PluginBackend          = pluginport.PluginBackend
	PluginCache            = pluginport.PluginCache
	PluginCommand          = pluginport.PluginCommand
	PluginCommandArgument  = pluginport.PluginCommandArgument
	PluginContributes      = pluginport.PluginContributes
	PluginCommands         = pluginport.PluginCommands
	PluginDeclarations     = pluginport.PluginDeclarations
	PluginConfigurator     = pluginport.PluginConfigurator
	PluginDiscoverer       = pluginport.PluginDiscoverer
	PluginEventEmitter     = pluginport.PluginEventEmitter
	PluginFrontend         = pluginport.PluginFrontend
	PluginLifecycleManager = pluginport.PluginLifecycleManager
	PluginManager          = pluginport.PluginManager
	PluginManifest         = pluginport.PluginManifest
	PluginManifestSetting  = pluginport.PluginManifestSetting
	PluginNoteType         = pluginport.PluginNoteType
	PluginReader           = pluginport.PluginReader
	PluginRegistryManager  = pluginport.PluginRegistryManager
	PluginResolver         = pluginport.PluginResolver
	PluginSearch           = pluginport.PluginSearch
	PluginSearchIndex      = pluginport.PluginSearchIndex
	PluginPassageSearch    = pluginport.PluginPassageSearch
	PluginSearchPolicy     = pluginport.PluginSearchPolicy
	SecretDeclaration      = pluginport.SecretDeclaration
	PluginSettingsView     = pluginport.PluginSettingsView
	PluginSettingCondition = pluginport.PluginSettingCondition
	PluginSettingOption    = pluginport.PluginSettingOption
	PluginSettingOptions   = pluginport.PluginSettingOptions
	PluginSettingOptionsBy = pluginport.PluginSettingOptionsBy
	PluginStore            = pluginport.PluginStore
	PluginStores           = pluginport.PluginStores
	PluginTextHost         = pluginport.PluginTextHost
	PluginSearchHost       = pluginport.PluginSearchHost
	PluginTexts            = pluginport.PluginTexts
	PluginVault            = pluginport.PluginVault
	PluginVaultLifecycle   = pluginport.PluginVaultLifecycle
	PluginWindow           = pluginport.PluginWindow
	PluginWindowHost       = pluginport.PluginWindowHost
	PluginVaultList        = pluginport.PluginVaultList
	PluginVaultListHost    = pluginport.PluginVaultListHost
	PluginVaultReader      = pluginport.PluginVaultReader
	PluginVaultStore       = pluginport.PluginVaultStore
	PluginVaultWriter      = pluginport.PluginVaultWriter
	PluginWork             = pluginport.PluginWork
)

const (
	ActivationEventStartup      = pluginport.ActivationEventStartup
	ActivationEventVectorsReady = pluginport.ActivationEventVectorsReady
	ActivationEventFilePrefix   = pluginport.ActivationEventFilePrefix
	CommandArgumentBoolean      = pluginport.CommandArgumentBoolean
	CommandArgumentNumber       = pluginport.CommandArgumentNumber
	CommandArgumentVersion      = pluginport.CommandArgumentVersion
	CommandArgumentString       = pluginport.CommandArgumentString
	DependencyKindBinary        = pluginport.DependencyKindBinary
	DependencyKindDataset       = pluginport.DependencyKindDataset
	DependencyKindModel         = pluginport.DependencyKindModel
	DependencyKindRuntime       = pluginport.DependencyKindRuntime
	PermissionAIComplete        = pluginport.PermissionAIComplete
	PermissionIndexRead         = pluginport.PermissionIndexRead
	PermissionSearchPolicy      = pluginport.PermissionSearchPolicy
	PermissionNetworkAlias      = pluginport.PermissionNetworkAlias
	PermissionNetworkOutbound   = pluginport.PermissionNetworkOutbound
	PermissionVaultRead         = pluginport.PermissionVaultRead
	PermissionVaultManage       = pluginport.PermissionVaultManage
	PermissionWindowFocus       = pluginport.PermissionWindowFocus
	RuntimeONNX                 = pluginport.RuntimeONNX
	SettingTypeEnum             = pluginport.SettingTypeEnum
	SettingTypeFile             = pluginport.SettingTypeFile
	SettingTypeString           = pluginport.SettingTypeString
)

var (
	ErrDuplicateSettingKey    = pluginport.ErrDuplicateSettingKey
	ErrEmptyID                = pluginport.ErrEmptyID
	ErrEmptyName              = pluginport.ErrEmptyName
	ErrEmptySettingKey        = pluginport.ErrEmptySettingKey
	ErrEmptySettingTitle      = pluginport.ErrEmptySettingTitle
	ErrEmptyVersion           = pluginport.ErrEmptyVersion
	ErrEnumMissingOptions     = pluginport.ErrEnumMissingOptions
	ErrInvalidActivationEvent = pluginport.ErrInvalidActivationEvent
	ErrInvalidBackend         = pluginport.ErrInvalidBackend
	ErrInvalidCapability      = pluginport.ErrInvalidCapability
	ErrInvalidCommand         = pluginport.ErrInvalidCommand
	ErrInvalidCommandArgument = pluginport.ErrInvalidCommandArgument
	ErrInvalidDependency      = pluginport.ErrInvalidDependency
	ErrInvalidFrame           = pluginport.ErrInvalidFrame
	ErrInvalidID              = pluginport.ErrInvalidID
	ErrInvalidNoteType        = pluginport.ErrInvalidNoteType
	ErrInvalidPermission      = pluginport.ErrInvalidPermission
	ErrInvalidReader          = pluginport.ErrInvalidReader
	ErrInvalidSettingsView    = pluginport.ErrInvalidSettingsView
	ErrInvalidSecret          = pluginport.ErrInvalidSecret
	ErrInvalidService         = pluginport.ErrInvalidService
	ErrInvalidSetting         = pluginport.ErrInvalidSetting
	ErrInvalidSettingKey      = pluginport.ErrInvalidSettingKey
	ErrInvalidSettingType     = pluginport.ErrInvalidSettingType
	ErrInvalidVersion         = pluginport.ErrInvalidVersion
	ErrNoBackend              = pluginport.ErrNoBackend
	ErrRangeInvalidBounds     = pluginport.ErrRangeInvalidBounds
	ErrReservedSettingKey     = pluginport.ErrReservedSettingKey
	ErrUnsupportedProtocol    = pluginport.ErrUnsupportedProtocol
)

func FormatSettingValue(value any) string { return pluginport.FormatSettingValue(value) }

func GetDependencyKinds() []manifest.DependencyKind { return pluginport.GetDependencyKinds() }

func NormalizePermission(permission string) string { return pluginport.NormalizePermission(permission) }

func PluginActivationEvents() []string { return pluginport.PluginActivationEvents() }

func PluginCapabilities() []string { return pluginport.PluginCapabilities() }

func PluginPermissions() []string { return pluginport.PluginPermissions() }
