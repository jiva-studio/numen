package bridge

import (
	"context"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// AbsentRuntime is the runtime of a plugin host that runs nothing: every plugin is unavailable for Reason, with
// Cause alongside, and what is set on it is accepted and kept nowhere.
type AbsentRuntime struct {
	Reason plugindomain.UnavailableReason
	Cause  error
}

func (r AbsentRuntime) unavailable(capability, pluginID string) error {
	return &plugindomain.UnavailableError{Capability: capability, PluginID: pluginID, Reason: r.Reason, Err: r.Cause}
}

// EnsureActive reports that the plugin is unavailable.
func (r AbsentRuntime) EnsureActive(_ context.Context, pluginID string) (transport.Endpoint, error) {
	return transport.Endpoint{}, r.unavailable("", pluginID)
}

// RetryPlugin reports that the plugin is unavailable.
func (r AbsentRuntime) RetryPlugin(_ context.Context, pluginID string) (*plugindomain.Failure, error) {
	return nil, r.unavailable("", pluginID)
}

// ResolveCapability reports that no plugin answers for the capability.
func (r AbsentRuntime) ResolveCapability(capability string) (manifest.Discovered, error) {
	return manifest.Discovered{}, r.unavailable(capability, "")
}

// ResolveService finds no plugin.
func (AbsentRuntime) ResolveService(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

func (AbsentRuntime) GetPlugin(string) (manifest.Discovered, bool) {
	return manifest.Discovered{}, false
}

// ResolveNoteProfile finds no plugin that defines the type.
func (AbsentRuntime) ResolveNoteProfile(domain.NoteType) (domain.NoteProfile, bool) {
	return domain.NoteProfile{}, false
}

// Release has no plugin to release.
func (AbsentRuntime) Release(string) {}

// Discover finds no plugin.
func (AbsentRuntime) Discover() error { return nil }

// GetPluginInfos lists no plugin.
func (AbsentRuntime) GetPluginInfos() []plugindomain.Info { return nil }

// IsPluginEnabled reports that no plugin is enabled.
func (AbsentRuntime) IsPluginEnabled(string) bool { return false }

// IsEnabled reports that no plugin is enabled.
func (AbsentRuntime) IsEnabled(string) bool { return false }

// EmitEvent has no plugin to tell.
func (AbsentRuntime) EmitEvent(string) {}

// SetSafeMode is accepted and ignored.
func (AbsentRuntime) SetSafeMode(bool) {}

// SetVaultPath is accepted and ignored.
func (AbsentRuntime) SetVaultPath(string) {}

// SetOnPluginFailed is accepted and ignored: no plugin fails.
func (AbsentRuntime) SetOnPluginFailed(func(plugindomain.Failure)) {}

// SetOnFailureCleared is accepted and ignored: no failure is cleared.
func (AbsentRuntime) SetOnFailureCleared(func(string)) {}

// IsPluginSuspended reports that no plugin is suspended.
func (AbsentRuntime) IsPluginSuspended(string) bool { return false }

// WatchPluginConfig never calls fn, and returns a stop that does nothing.
func (AbsentRuntime) WatchPluginConfig(string, func()) func() { return func() {} }

// AbsentSettings is the configuration and the watch of a plugin host that runs nothing: they accept and keep
// nothing.
type AbsentSettings struct{}

func (AbsentSettings) GetPluginSettingDeclarations(string) ([]manifest.Setting, bool) {
	return nil, false
}

// SetKeyStore is accepted and ignored.
func (AbsentSettings) SetKeyStore(pluginport.KeyStore) {}

// SetSecretSource is accepted and ignored.
func (AbsentSettings) SetSecretSource(plugindomain.SecretSource) {}

// ListDeclarations returns no declaration.
func (AbsentSettings) ListDeclarations() []plugindomain.DeclarationKey { return nil }

// Refresh configures no plugin.
func (AbsentSettings) Refresh(context.Context, ...string) error { return nil }

// SetSettings is accepted and ignored.
func (AbsentSettings) SetSettings(map[string]map[string]any) {}

// SetPerformanceProfile is accepted and ignored.
func (AbsentSettings) SetPerformanceProfile(context.Context, string) {}

// CheckPluginSetting accepts every value: no plugin declares the setting.
func (AbsentSettings) CheckPluginSetting(string, string, any) error { return nil }

// SetPersistHandler is accepted and ignored.
func (AbsentSettings) SetPersistHandler(plugindomain.PersistHandler) {}

// TeardownAll has no plugin to stop.
func (AbsentSettings) TeardownAll(context.Context, time.Duration) error { return nil }

// ObserveDependencies is accepted and ignored: no plugin declares a dependency.
func (AbsentSettings) ObserveDependencies(func(manifest.DependencyRequirement, plugindomain.DependencyStatus)) {
}
