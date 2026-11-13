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

// PluginRetrier starts a plugin once, as any use of it does.
type PluginRetrier interface {
	// RetryPlugin answers why the start failed where it did.
	RetryPlugin(ctx context.Context, pluginID string) (*plugin.Failure, error)
}

// PluginDiscoverer scans for available plugins.
type PluginDiscoverer interface {
	Discover() error
}

// PluginLister returns the metadata of the plugins in the order of their ids.
type PluginLister interface {
	GetPluginInfos() []plugin.Info
	// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
	IsPluginEnabled(pluginID string) bool
}

type PluginSettingDeclarer interface {
	// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and
	// false for a plugin the host does not know.
	GetPluginSettingDeclarations(pluginID string) ([]manifest.Setting, bool)
}

// ConfigRefresher sends the configuration again to the plugins that were not given it yet.
type ConfigRefresher interface {
	// Refresh configures the running plugins that were not given their configuration yet, and the
	// plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
}

// SettingsPush gives the plugins the settings the user keeps for them.
type SettingsPush interface {
	SetSettings(all map[string]map[string]any)
	SetPerformanceProfile(ctx context.Context, profile string)
}

// ConfigWatcher tells when a plugin has been given its configuration, and which configuration it is.
type ConfigWatcher interface {
	WatchPluginConfig(pluginID string, fn func()) (stop func())
	// GetPluginConfigRevision returns a digest of the version, the settings and the performance profile the
	// plugin is configured with. Equal revisions are one configuration, in this process or another.
	GetPluginConfigRevision(pluginID string) string
}

// RunningPlugins tells which plugins have a process that runs.
type RunningPlugins interface {
	// HasEndpoint reports whether the plugin has the endpoint of a running process.
	HasEndpoint(pluginID string) bool
}

type SettingChecker interface {
	CheckPluginSetting(pluginID, key string, value any) error
}

// PersistHook receives the function that saves a setting a plugin changes.
type PersistHook interface {
	SetPersistHandler(handler plugin.PersistHandler)
}

type PluginEventEmitter interface {
	EmitEvent(event string)
}

// PluginFailures tells the plugins whose start failed.
type PluginFailures interface {
	SetOnFailureCleared(fn func(pluginID string))
	// IsPluginSuspended reports whether the last start of the plugin failed, which holds it suspended.
	IsPluginSuspended(pluginID string) bool
}

// PluginShutdown stops the plugins when the host shuts down.
type PluginShutdown interface {
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}

// PluginBinding gives the host the mode and the vault path the plugins run under.
type PluginBinding interface {
	SetSafeMode(enabled bool)
	SetVaultPath(path string)
}
