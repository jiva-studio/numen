package pluginport

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

var (
	// ErrUnavailable means whatever answers cannot be reached right now.
	ErrUnavailable = domain.ErrUnavailable
)

// UnavailableReason is why no plugin answers.
type UnavailableReason int

const (
	// PluginNotInstalled is no installed plugin to answer.
	PluginNotInstalled UnavailableReason = iota + 1
	// PluginDisabled is every plugin that would answer switched off, by its setting or by safe mode.
	PluginDisabled
	// PluginFailed is a plugin that could not be started or reached.
	PluginFailed
	// PluginWithoutBackend is a plugin that declares no backend, so there is nothing to start.
	PluginWithoutBackend
)

// UnavailableError is a plugin, or a capability, that no plugin answers for, and why. It is
// ErrUnavailable.
type UnavailableError struct {
	Capability string
	PluginID   string
	Reason     UnavailableReason
	Err        error
}

// Error reports what is unavailable and why.
func (e *UnavailableError) Error() string {
	subject := "plugin " + e.PluginID
	if e.PluginID == "" {
		subject = "the plugin for " + e.Capability
	}
	var response string
	switch e.Reason {
	case PluginNotInstalled:
		response = subject + " is not installed"
	case PluginDisabled:
		response = subject + " is disabled"
	case PluginWithoutBackend:
		response = subject + " has no backend"
	default:
		response = subject + " failed to start"
	}
	if e.Err != nil {
		return response + ": " + e.Err.Error()
	}
	return response
}

// Is reports that the error is ErrUnavailable.
func (e *UnavailableError) Is(target error) bool {
	return target == ErrUnavailable
}

// Unwrap returns the cause the host gave.
func (e *UnavailableError) Unwrap() error {
	return e.Err
}

// PluginFailure is a plugin the host suspends after it failed to start, and why.
type PluginFailure struct {
	PluginID string
	Name     string
	Reason   string
	Detail   string
	At       time.Time
}

// PluginPersistHandler is called when a plugin modifies its own settings.
type PluginPersistHandler func(pluginID string, key string, value any) error

// PluginTransport represents an underlying communication channel for a plugin endpoint.
type PluginTransport interface {
	RoundTripTransport()
}

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host      string
	Port      int
	Token     string
	Transport PluginTransport
}

// Address returns host and port joined with a colon.
func (e Endpoint) Address() string {
	if e.Port == 0 {
		return e.Host
	}
	return fmt.Sprintf("%s:%d", e.Host, e.Port)
}

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
	EnsureActive(ctx context.Context, pluginID string) (Endpoint, error)
	Release(pluginID string)
}

// PluginDiscoverer scans for available plugins and returns their metadata in the order of their ids.
type PluginDiscoverer interface {
	Discover() error
	GetPluginInfos() []PluginInfo
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
	SetSecretSource(source SecretSource)
	// Refresh configures the running plugins whose list of secrets differs from the one they were
	// last given, and the plugins named in forced.
	Refresh(ctx context.Context, forced ...string) error
	ListDeclarations() []DeclarationKey
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
	SetPersistHandler(handler PluginPersistHandler)
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
	CheckPluginSetting(pluginID, key string, value any) error
}

// PluginEventEmitter broadcasts events to plugins.
type PluginEventEmitter interface {
	EmitEvent(event string)
}

type pluginFailureCallbacks interface {
	SetOnPluginFailed(fn func(PluginFailure))
	SetOnFailureCleared(fn func(pluginID string))
	RetryPlugin(ctx context.Context, pluginID string) (*PluginFailure, error)
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
