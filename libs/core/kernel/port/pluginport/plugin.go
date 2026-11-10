package pluginport

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// PluginResolver locates plugins by capability or identifier. Of several enabled plugins that
// declare a capability, the one with the lowest id is found.
type PluginResolver interface {
	// ResolveCapability returns the plugin for the capability. Where none answers, the error is an
	// *UnavailableError saying whether none is installed or every one is disabled.
	ResolveCapability(capability string) (DiscoveredPlugin, error)
	// ResolveService returns the enabled plugin with the lowest id among those that serve the service
	// to the windows, and false where none does.
	ResolveService(service string) (DiscoveredPlugin, bool)
	// ResolveNoteProfile returns how the index keeps the notes of a type, as the enabled plugin that
	// declares the type defines, and false where none does.
	ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool)
	GetPlugin(id string) (DiscoveredPlugin, bool)
}

// PluginActivator activates plugins and manages active references.
type PluginActivator interface {
	EnsureActive(ctx context.Context, pluginID string) (plugin.Endpoint, error)
	Release(pluginID string)
}

// PluginDiscoverer scans for available plugins and returns their metadata in the order of their ids.
type PluginDiscoverer interface {
	Discover() error
	GetPluginInfos() []plugin.Info
	// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
	IsPluginEnabled(pluginID string) bool
	// IsPluginSuspended reports whether the last start of the plugin failed, which holds it suspended.
	IsPluginSuspended(pluginID string) bool
}

// PluginSettingDeclarer returns the settings a plugin declares.
type PluginSettingDeclarer interface {
	// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and
	// false for a plugin the host does not know.
	GetPluginSettingDeclarations(pluginID string) ([]PluginManifestSetting, bool)
}

type pluginSecretConfigurator interface {
	SetKeyStore(store KeyStore)
	SetSecretSource(source plugin.SecretSource)
	// Refresh configures the running plugins whose list of secrets differs from the one they were
	// last given, and the plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
	ListDeclarations() []plugin.DeclarationKey
}

type pluginSettingsPush interface {
	SetSettings(all map[string]map[string]any)
	SetPerformanceProfile(ctx context.Context, profile string)
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}

// PluginConfigurator manages configuration and settings for plugins.
type PluginConfigurator interface {
	pluginSecretConfigurator
	pluginSettingsPush
	SetPersistHandler(handler plugin.PersistHandler)
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
	CheckPluginSetting(pluginID, key string, value any) error
}

// PluginEventEmitter broadcasts events to plugins.
type PluginEventEmitter interface {
	EmitEvent(event string)
}

type pluginFailureCallbacks interface {
	SetOnPluginFailed(fn func(plugin.Failure))
	SetOnFailureCleared(fn func(pluginID string))
	RetryPlugin(ctx context.Context, pluginID string) (*plugin.Failure, error)
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}

type pluginVaultBinding interface {
	SetSafeMode(enabled bool)
	SetPluginVault(vault PluginVault)
	SetPluginWork(work PluginWork)
	SetVaultPath(path string)
}

// PluginLifecycleManager coordinates process lifecycle, failure handling, and teardown for plugins.
type PluginLifecycleManager interface {
	PluginActivator
	pluginVaultBinding
	pluginFailureCallbacks
}

// PluginRegistryManager coordinates discovery, configuration, resolution, and capabilities for plugins.
type PluginRegistryManager interface {
	PluginDiscoverer
	PluginResolver
	PluginConfigurator
	PluginSettingDeclarer
	PluginEventEmitter
}

type pluginContentTypes interface {
	GetContentType(name string) domain.ContentType
	ListExtensions(mask domain.ContentType) []string
}

// pluginHostServices receives what the host lends the plugins that call it.
type pluginHostServices interface {
	PluginTextHost
	PluginSearchHost
	PluginWindowHost
	PluginVaultListHost
}

// PluginManager coordinates process lifecycle, discovery, settings, and RPC routing for modular extensions.
type PluginManager interface {
	PluginLifecycleManager
	PluginRegistryManager
	pluginHostServices
	pluginContentTypes
	GetDependencies() DependencyCatalog
}
