// Package plugincontract re-exports the plugin port contract.
package plugincontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
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
	PluginCommandEntry     = pluginport.PluginCommandEntry
	PluginSlotEntry        = pluginport.PluginSlotEntry
	PluginKeybindingEntry  = pluginport.PluginKeybindingEntry
	PluginTabKindEntry     = pluginport.PluginTabKindEntry
	PluginTriggerEntry     = pluginport.PluginTriggerEntry
	PluginContributes      = pluginport.PluginContributes
	PluginCommands         = pluginport.PluginCommands
	PluginDeclarations     = pluginport.PluginDeclarations
	PluginConfigurator     = pluginport.PluginConfigurator
	PluginDiscoverer       = pluginport.PluginDiscoverer
	PluginEmbeddedTexts    = pluginport.PluginEmbeddedTexts
	PluginEmbeddingState   = pluginport.PluginEmbeddingState
	PluginFileRange        = pluginport.PluginFileRange
	PluginExtensionStats   = pluginport.PluginExtensionStats
	PluginIndexStats       = pluginport.PluginIndexStats
	PluginEventEmitter     = pluginport.PluginEventEmitter
	PluginFloorSource      = pluginport.PluginFloorSource
	PluginFrontend         = pluginport.PluginFrontend
	PluginIndexState       = pluginport.PluginIndexState
	PluginIndexStatus      = pluginport.PluginIndexStatus
	PluginInfo             = pluginport.PluginInfo
	PluginLinkAdded        = pluginport.PluginLinkAdded
	PluginLinkAddition     = pluginport.PluginLinkAddition
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
	PluginPassageQuery     = pluginport.PluginPassageQuery
	PluginPassageSearch    = pluginport.PluginPassageSearch
	PluginSearchPolicy     = pluginport.PluginSearchPolicy
	PluginSearchFloor      = pluginport.PluginSearchFloor
	PluginSimilarityQuery  = pluginport.PluginSimilarityQuery
	PluginSimilaritySample = pluginport.PluginSimilaritySample
	SecretDeclaration      = pluginport.SecretDeclaration
	PluginSettingsView     = pluginport.PluginSettingsView
	PluginSetting          = pluginport.PluginSetting
	PluginSettingCondition = pluginport.PluginSettingCondition
	PluginSettingOption    = pluginport.PluginSettingOption
	PluginSettingOptions   = pluginport.PluginSettingOptions
	PluginSettingOptionsBy = pluginport.PluginSettingOptionsBy
	PluginStore            = pluginport.PluginStore
	PluginStoreFile        = pluginport.PluginStoreFile
	PluginStores           = pluginport.PluginStores
	PluginTask             = pluginport.PluginTask
	PluginTextHost         = pluginport.PluginTextHost
	PluginSearchHost       = pluginport.PluginSearchHost
	PluginTexts            = pluginport.PluginTexts
	PluginTextListing      = pluginport.PluginTextListing
	PluginTextPage         = pluginport.PluginTextPage
	PluginTextRange        = pluginport.PluginTextRange
	PluginTextPut          = pluginport.PluginTextPut
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
	TaskUnit               = pluginport.TaskUnit
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
	MaxSampledQueries           = pluginport.MaxSampledQueries
	MaxSampledPassages          = pluginport.MaxSampledPassages
	PluginFloorDefault          = pluginport.PluginFloorDefault
	PluginFloorModelHint        = pluginport.PluginFloorModelHint
	PluginFloorStored           = pluginport.PluginFloorStored
	PluginIndexFailed           = pluginport.PluginIndexFailed
	PluginIndexed               = pluginport.PluginIndexed
	PluginIndexing              = pluginport.PluginIndexing
	RuntimeONNX                 = pluginport.RuntimeONNX
	SettingTypeEnum             = pluginport.SettingTypeEnum
	SettingTypeFile             = pluginport.SettingTypeFile
	SettingTypeString           = pluginport.SettingTypeString
	TaskUnitBytes               = pluginport.TaskUnitBytes
	TaskUnitSeconds             = pluginport.TaskUnitSeconds
	TaskUnitThings              = pluginport.TaskUnitThings
)

var (
	ErrDuplicateSettingKey    = pluginport.ErrDuplicateSettingKey
	ErrEmptyID                = pluginport.ErrEmptyID
	ErrEmptyName              = pluginport.ErrEmptyName
	ErrEmptySettingKey        = pluginport.ErrEmptySettingKey
	ErrEmptySettingTitle      = pluginport.ErrEmptySettingTitle
	ErrEmptyVersion           = pluginport.ErrEmptyVersion
	ErrEmbeddingFailed        = plugin.ErrEmbeddingFailed
	ErrInvalidQuery           = plugin.ErrInvalidQuery
	ErrEnumMissingOptions     = pluginport.ErrEnumMissingOptions
	ErrInvalidActivationEvent = pluginport.ErrInvalidActivationEvent
	ErrInvalidBackend         = pluginport.ErrInvalidBackend
	ErrInvalidCapability      = pluginport.ErrInvalidCapability
	ErrInvalidCommand         = pluginport.ErrInvalidCommand
	ErrInvalidCommandArgument = pluginport.ErrInvalidCommandArgument
	ErrInvalidDependency      = pluginport.ErrInvalidDependency
	ErrInvalidFrame           = pluginport.ErrInvalidFrame
	ErrInvalidFloor           = plugin.ErrInvalidFloor
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
	ErrLimitExceeded          = plugin.ErrLimitExceeded
	ErrNoBackend              = pluginport.ErrNoBackend
	ErrNoModel                = plugin.ErrNoModel
	ErrNoKey                  = plugin.ErrNoKey
	ErrNoKeyStore             = plugin.ErrNoKeyStore
	ErrNoProvisioner          = plugin.ErrNoProvisioner
	ErrNoWindow               = plugin.ErrNoWindow
	ErrNotText                = plugin.ErrNotText
	ErrUnsavedChanges         = plugin.ErrUnsavedChanges
	ErrInvalidFocus           = plugin.ErrInvalidFocus
	ErrTabNotShown            = plugin.ErrTabNotShown
	ErrPathTraversal          = plugin.ErrPathTraversal
	ErrPermissionDenied       = plugin.ErrPermissionDenied
	ErrRangeInvalidBounds     = pluginport.ErrRangeInvalidBounds
	ErrRecipeMismatch         = plugin.ErrRecipeMismatch
	ErrReservedSettingKey     = pluginport.ErrReservedSettingKey
	ErrUnsupportedProtocol    = pluginport.ErrUnsupportedProtocol
)

func FormatSettingValue(value any) string { return pluginport.FormatSettingValue(value) }

func GetDependencyKinds() []manifest.DependencyKind { return pluginport.GetDependencyKinds() }

func NormalizePermission(permission string) string { return pluginport.NormalizePermission(permission) }

func PluginActivationEvents() []string { return pluginport.PluginActivationEvents() }

func PluginCapabilities() []string { return pluginport.PluginCapabilities() }

func PluginPermissions() []string { return pluginport.PluginPermissions() }

func NewPluginSetting(declared PluginManifestSetting, defaultJSON string, options PluginSettingOptions) PluginSetting {
	return pluginport.NewPluginSetting(declared, defaultJSON, options)
}
