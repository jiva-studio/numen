// Package plugincontract re-exports the plugin port contract.
package plugincontract

import (
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

type (
	CommandFailedError     = pluginport.CommandFailedError
	DependencyCatalog      = pluginport.DependencyCatalog
	DependencyKind         = pluginport.DependencyKind
	DependencyProvisioner  = pluginport.DependencyProvisioner
	DependencyRequirement  = pluginport.DependencyRequirement
	DependencyState        = pluginport.DependencyState
	DependencyStatus       = pluginport.DependencyStatus
	DiscoveredPlugin       = pluginport.DiscoveredPlugin
	Endpoint               = pluginport.Endpoint
	KeyStore               = pluginport.KeyStore
	Lease                  = pluginport.Lease
	PluginActivator        = pluginport.PluginActivator
	PluginBackend          = pluginport.PluginBackend
	PluginCache            = pluginport.PluginCache
	PluginCommand          = pluginport.PluginCommand
	PluginCommandArgument  = pluginport.PluginCommandArgument
	PluginCommandCall      = pluginport.PluginCommandCall
	PluginCommandEntry     = pluginport.PluginCommandEntry
	PluginCommandResult    = pluginport.PluginCommandResult
	PluginMenuEntry        = pluginport.PluginMenuEntry
	PluginContextCondition = pluginport.PluginContextCondition
	PluginContext          = pluginport.PluginContext
	PluginContributes      = pluginport.PluginContributes
	PluginCommands         = pluginport.PluginCommands
	PluginParts            = pluginport.PluginParts
	PluginConfigurator     = pluginport.PluginConfigurator
	PluginDiscoverer       = pluginport.PluginDiscoverer
	PluginEmbeddedTexts    = pluginport.PluginEmbeddedTexts
	PluginEmbeddingState   = pluginport.PluginEmbeddingState
	PluginExtensionStats   = pluginport.PluginExtensionStats
	PluginIndexStats       = pluginport.PluginIndexStats
	PluginEventEmitter     = pluginport.PluginEventEmitter
	PluginFailure          = pluginport.PluginFailure
	PluginFloorSource      = pluginport.PluginFloorSource
	PluginFrontend         = pluginport.PluginFrontend
	PluginIndexState       = pluginport.PluginIndexState
	PluginIndexStatus      = pluginport.PluginIndexStatus
	PluginInfo             = pluginport.PluginInfo
	PluginLifecycleManager = pluginport.PluginLifecycleManager
	PluginManager          = pluginport.PluginManager
	PluginManifest         = pluginport.PluginManifest
	PluginManifestSetting  = pluginport.PluginManifestSetting
	PluginNoteType         = pluginport.PluginNoteType
	PluginPersistHandler   = pluginport.PluginPersistHandler
	PluginHandle           = pluginport.PluginHandle
	PluginReader           = pluginport.PluginReader
	PluginReaderInfo       = pluginport.PluginReaderInfo
	PluginRegistryManager  = pluginport.PluginRegistryManager
	PluginResolver         = pluginport.PluginResolver
	PluginSearch           = pluginport.PluginSearch
	PluginSearchIndex      = pluginport.PluginSearchIndex
	PluginSearchPolicy     = pluginport.PluginSearchPolicy
	PluginSearchFloor      = pluginport.PluginSearchFloor
	PluginSimilarityQuery  = pluginport.PluginSimilarityQuery
	PluginSimilaritySample = pluginport.PluginSimilaritySample
	PluginSecret           = pluginport.PluginSecret
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
	PluginTextPut          = pluginport.PluginTextPut
	PluginTransport        = pluginport.PluginTransport
	PluginVault            = pluginport.PluginVault
	PluginVaultLifecycle   = pluginport.PluginVaultLifecycle
	PluginVaultReader      = pluginport.PluginVaultReader
	PluginVaultStore       = pluginport.PluginVaultStore
	PluginVaultWriter      = pluginport.PluginVaultWriter
	PluginWork             = pluginport.PluginWork
	ResolvedDependency     = pluginport.ResolvedDependency
	TaskUnit               = pluginport.TaskUnit
	UnavailableError       = pluginport.UnavailableError
	UnavailableReason      = pluginport.UnavailableReason
)

const (
	ActivationEventAll          = pluginport.ActivationEventAll
	CommandArgumentBoolean      = pluginport.CommandArgumentBoolean
	CommandArgumentNumber       = pluginport.CommandArgumentNumber
	CommandArgumentVersion      = pluginport.CommandArgumentVersion
	CommandArgumentString       = pluginport.CommandArgumentString
	DependencyKindBinary        = pluginport.DependencyKindBinary
	DependencyKindDataset       = pluginport.DependencyKindDataset
	DependencyKindModel         = pluginport.DependencyKindModel
	DependencyKindRuntime       = pluginport.DependencyKindRuntime
	DependencyStateFailed       = pluginport.DependencyStateFailed
	DependencyStateMissing      = pluginport.DependencyStateMissing
	DependencyStateProvisioning = pluginport.DependencyStateProvisioning
	DependencyStateReady        = pluginport.DependencyStateReady
	PermissionAIComplete        = pluginport.PermissionAIComplete
	PermissionIndexRead         = pluginport.PermissionIndexRead
	PermissionSearchPolicy      = pluginport.PermissionSearchPolicy
	PermissionNetworkAlias      = pluginport.PermissionNetworkAlias
	PermissionNetworkOutbound   = pluginport.PermissionNetworkOutbound
	PermissionVaultRead         = pluginport.PermissionVaultRead
	PermissionVaultWrite        = pluginport.PermissionVaultWrite
	MaxEmbeddedBytes            = pluginport.MaxEmbeddedBytes
	MaxEmbeddedTexts            = pluginport.MaxEmbeddedTexts
	MaxSampledQueries           = pluginport.MaxSampledQueries
	MaxSampledPassages          = pluginport.MaxSampledPassages
	PluginFloorDefault          = pluginport.PluginFloorDefault
	PluginFloorModelHint        = pluginport.PluginFloorModelHint
	PluginFloorStored           = pluginport.PluginFloorStored
	PluginDisabled              = pluginport.PluginDisabled
	PluginFailed                = pluginport.PluginFailed
	PluginIndexFailed           = pluginport.PluginIndexFailed
	PluginIndexed               = pluginport.PluginIndexed
	PluginIndexing              = pluginport.PluginIndexing
	PluginNotInstalled          = pluginport.PluginNotInstalled
	PluginProtocolVersion       = pluginport.PluginProtocolVersion
	PluginWithoutBackend        = pluginport.PluginWithoutBackend
	RuntimeONNX                 = pluginport.RuntimeONNX
	HandlePlaybackCopy          = pluginport.HandlePlaybackCopy
	HandleTextProvider          = pluginport.HandleTextProvider
	ContextAnyValue             = pluginport.ContextAnyValue
	ConfigPerformanceProfile    = pluginport.ConfigPerformanceProfile
	SettingStoredSecrets        = pluginport.SettingStoredSecrets
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
	ErrEmbeddingFailed        = pluginport.ErrEmbeddingFailed
	ErrInvalidQuery           = pluginport.ErrInvalidQuery
	ErrEnumMissingOptions     = pluginport.ErrEnumMissingOptions
	ErrInvalidActivationEvent = pluginport.ErrInvalidActivationEvent
	ErrInvalidBackend         = pluginport.ErrInvalidBackend
	ErrInvalidCapability      = pluginport.ErrInvalidCapability
	ErrInvalidCommand         = pluginport.ErrInvalidCommand
	ErrInvalidCommandArgument = pluginport.ErrInvalidCommandArgument
	ErrInvalidDependency      = pluginport.ErrInvalidDependency
	ErrInvalidFrame           = pluginport.ErrInvalidFrame
	ErrInvalidFloor           = pluginport.ErrInvalidFloor
	ErrInvalidID              = pluginport.ErrInvalidID
	ErrInvalidNoteType        = pluginport.ErrInvalidNoteType
	ErrInvalidPermission      = pluginport.ErrInvalidPermission
	ErrInvalidHandle          = pluginport.ErrInvalidHandle
	ErrInvalidReader          = pluginport.ErrInvalidReader
	ErrInvalidSettingsView    = pluginport.ErrInvalidSettingsView
	ErrInvalidSecret          = pluginport.ErrInvalidSecret
	ErrInvalidService         = pluginport.ErrInvalidService
	ErrInvalidSetting         = pluginport.ErrInvalidSetting
	ErrInvalidSettingKey      = pluginport.ErrInvalidSettingKey
	ErrInvalidSettingType     = pluginport.ErrInvalidSettingType
	ErrInvalidVersion         = pluginport.ErrInvalidVersion
	ErrLimitExceeded          = pluginport.ErrLimitExceeded
	ErrNoBackend              = pluginport.ErrNoBackend
	ErrNoModel                = pluginport.ErrNoModel
	ErrNoKey                  = pluginport.ErrNoKey
	ErrNoKeyStore             = pluginport.ErrNoKeyStore
	ErrNoProvisioner          = pluginport.ErrNoProvisioner
	ErrPathTraversal          = pluginport.ErrPathTraversal
	ErrPermissionDenied       = pluginport.ErrPermissionDenied
	ErrRangeInvalidBounds     = pluginport.ErrRangeInvalidBounds
	ErrRecipeMismatch         = pluginport.ErrRecipeMismatch
	ErrReservedSettingKey     = pluginport.ErrReservedSettingKey
	ErrUnavailable            = pluginport.ErrUnavailable
	ErrUnsupportedProtocol    = pluginport.ErrUnsupportedProtocol
)

func FormatSettingValue(value any) string { return pluginport.FormatSettingValue(value) }

func GetDependencyKinds() []manifest.DependencyKind { return pluginport.GetDependencyKinds() }

func NormalizePermission(permission string) string { return pluginport.NormalizePermission(permission) }

func PluginCapabilities() []string { return pluginport.PluginCapabilities() }

func PluginPermissions() []string { return pluginport.PluginPermissions() }

func NewPluginSetting(declared PluginManifestSetting, defaultJSON string, options PluginSettingOptions) PluginSetting {
	return pluginport.NewPluginSetting(declared, defaultJSON, options)
}
