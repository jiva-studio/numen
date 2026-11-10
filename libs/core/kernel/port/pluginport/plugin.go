package pluginport

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginResolver locates plugins by capability or identifier. Of several enabled plugins that
// declare a capability, the one with the lowest id is found.
type PluginResolver interface {
	// ResolveCapability returns the plugin for the capability. Where none answers, the error is an
	// *UnavailableError saying whether none is installed or every one is disabled.
	ResolveCapability(capability string) (manifest.Discovered, error)
	// ResolveService returns the enabled plugin with the lowest id among those that serve the service
	// to the windows, and false where none does.
	ResolveService(service string) (manifest.Discovered, bool)
	// ResolveNoteProfile returns how the index keeps the notes of a type, as the enabled plugin that
	// declares the type defines, and false where none does.
	ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool)
	GetPlugin(id string) (manifest.Discovered, bool)
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
	GetPluginSettingDeclarations(pluginID string) ([]manifest.Setting, bool)
}

// SecretConfigurator hands the secrets to the plugins that declare them.
type SecretConfigurator interface {
	SetKeyStore(store KeyStore)
	SetSecretSource(source plugin.SecretSource)
	// Refresh configures the running plugins whose list of secrets differs from the one they were
	// last given, and the plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
	ListDeclarations() []plugin.DeclarationKey
}

// SettingsPush gives the plugins the settings the user keeps for them.
type SettingsPush interface {
	SetSettings(all map[string]map[string]any)
	SetPerformanceProfile(ctx context.Context, profile string)
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}

// SettingUpdater changes and checks one setting of a plugin.
type SettingUpdater interface {
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
	CheckPluginSetting(pluginID, key string, value any) error
}

// PersistHook receives the function that saves a setting a plugin changes.
type PersistHook interface {
	SetPersistHandler(handler plugin.PersistHandler)
}

// PluginEventEmitter broadcasts events to plugins.
type PluginEventEmitter interface {
	EmitEvent(event string)
}

// PluginFailures reports the plugins whose start failed and restarts them.
type PluginFailures interface {
	SetOnPluginFailed(fn func(plugin.Failure))
	SetOnFailureCleared(fn func(pluginID string))
	RetryPlugin(ctx context.Context, pluginID string) (*plugin.Failure, error)
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}

// PluginBinding gives the host the mode and the vault path the plugins run under.
type PluginBinding interface {
	SetSafeMode(enabled bool)
	SetVaultPath(path string)
}
