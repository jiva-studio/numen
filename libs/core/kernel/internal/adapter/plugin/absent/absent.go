package absent

import (
	"context"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Runtime is the runtime of a plugin host that runs nothing: every plugin is unavailable for Reason, with
// Cause alongside, and what is set on it is accepted and kept nowhere.
type Runtime struct {
	Reason plugindomain.UnavailableReason
	Cause  error
}

func (r Runtime) unavailable(capability, pluginID string) error {
	return &plugindomain.UnavailableError{Capability: capability, PluginID: pluginID, Reason: r.Reason, Err: r.Cause}
}

// EnsureActive reports that the plugin is unavailable.
func (r Runtime) EnsureActive(_ context.Context, pluginID string) (transport.Endpoint, error) {
	return transport.Endpoint{}, r.unavailable("", pluginID)
}

// RetryPlugin reports that the plugin is unavailable.
func (r Runtime) RetryPlugin(_ context.Context, pluginID string) (*plugindomain.Failure, error) {
	return nil, r.unavailable("", pluginID)
}

// ResolveCapability reports that no plugin answers for the capability.
func (r Runtime) ResolveCapability(capability string) (manifest.Discovered, error) {
	return manifest.Discovered{}, r.unavailable(capability, "")
}

// ResolveService finds no plugin.
func (Runtime) ResolveService(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

func (Runtime) GetPlugin(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

// ResolveNoteProfile finds no plugin that defines the type.
func (Runtime) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

// Release has no plugin to release.
func (Runtime) Release(string) {}

// Discover finds no plugin.
func (Runtime) Discover() error { return nil }

// GetPluginInfos lists no plugin.
func (Runtime) GetPluginInfos() []plugindomain.Info { return nil }

// IsPluginEnabled reports that no plugin is enabled.
func (Runtime) IsPluginEnabled(string) bool { return false }

// IsEnabled reports that no plugin is enabled.
func (Runtime) IsEnabled(string) bool { return false }

// EmitEvent has no plugin to tell.
func (Runtime) EmitEvent(string) {}

// SetOnFailureCleared is accepted and ignored: no failure is cleared.
func (Runtime) SetOnFailureCleared(func(string)) {}

// SetSafeMode is accepted and ignored.
func (Runtime) SetSafeMode(bool) {}

// SetVaultPath is accepted and ignored.
func (Runtime) SetVaultPath(string) {}

// IsPluginSuspended reports that no plugin is suspended.
func (Runtime) IsPluginSuspended(string) bool { return false }

// WatchPluginConfig never calls fn, and returns a stop that does nothing.
func (Runtime) WatchPluginConfig(string, func()) func() { return func() {} }

// Settings is the configuration and the watch of a plugin host that runs nothing: they accept and keep
// nothing.
type Settings struct{}

func (Settings) GetPluginSettingDeclarations(string) ([]manifest.Setting, bool) {
	return nil, false
}

// Refresh configures no plugin.
func (Settings) Refresh(context.Context, ...string) error { return nil }

// SetSettings is accepted and ignored.
func (Settings) SetSettings(map[string]map[string]any) {}

// SetPerformanceProfile is accepted and ignored.
func (Settings) SetPerformanceProfile(context.Context, string) {}

// CheckPluginSetting accepts every value: no plugin declares the setting.
func (Settings) CheckPluginSetting(string, string, any) error { return nil }

// SetPersistHandler is accepted and ignored.
func (Settings) SetPersistHandler(plugindomain.PersistHandler) {}

// TeardownAll has no plugin to stop.
func (Settings) TeardownAll(context.Context, time.Duration) error { return nil }

// ObserveDependencies is accepted and ignored: no plugin declares a dependency.
func (Settings) ObserveDependencies(func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
}

// Processes is the processes of a plugin host that runs nothing.
type Processes struct{}

// ListProcesses is empty: a host that runs nothing starts no process.
func (Processes) ListProcesses() []plugindomain.Process { return nil }

// WatchProcesses never signals: nothing runs to change.
func (Processes) WatchProcesses(context.Context) <-chan struct{} { return nil }
