package port

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	// ErrPathTraversal is returned when a path escapes the active vault boundary.
	ErrPathTraversal = errors.New("path traversal outside active vault is prohibited")

	// ErrPermissionDenied is returned when a plugin lacks the required permission.
	ErrPermissionDenied = errors.New("plugin permission denied")
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
)

// UnavailableError is a plugin, or a capability, that no plugin answers for, and why. It is
// ErrUnavailable.
type UnavailableError struct {
	// Capability is what was asked for. Empty where a plugin was asked for by its id.
	Capability string
	// PluginID is the plugin asked for or found. Empty where none was found.
	PluginID string
	Reason   UnavailableReason
	// Err is the cause the host gave.
	Err error
}

// Error says what is unavailable and why.
func (e *UnavailableError) Error() string {
	subject := "plugin " + e.PluginID
	if e.PluginID == "" {
		subject = "the plugin for " + e.Capability
	}
	var said string
	switch e.Reason {
	case PluginNotInstalled:
		said = subject + " is not installed"
	case PluginDisabled:
		said = subject + " is disabled"
	default:
		said = subject + " failed to start"
	}
	if e.Err != nil {
		return said + ": " + e.Err.Error()
	}
	return said
}

// Is reports that the error is ErrUnavailable.
func (e *UnavailableError) Is(target error) bool {
	return target == ErrUnavailable
}

// Unwrap returns the cause the host gave.
func (e *UnavailableError) Unwrap() error {
	return e.Err
}

// PluginFailure is a plugin the host holds off after it failed to start, and why.
type PluginFailure struct {
	PluginID string
	// Name is what the plugin is called to a person.
	Name string
	// Reason is why the plugin failed, in words a person reads.
	Reason string
	// At is when the host gave the plugin up.
	At time.Time
}

// PluginPersistHandler is called when a plugin modifies its own settings.
type PluginPersistHandler func(pluginID string, key string, value any) error

// DiscoveredPlugin is a validated plugin located on disk.
type DiscoveredPlugin struct {
	Dir      string
	Manifest PluginManifest
}

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

// BaseURL returns the HTTP scheme, host, and port for the plugin server.
func (e Endpoint) BaseURL() string {
	if e.Port == 0 {
		if e.Host != "" {
			return fmt.Sprintf("http://%s", e.Host)
		}
		return "http://inprocess"
	}
	return fmt.Sprintf("http://%s:%d", e.Host, e.Port)
}

// PluginResolver locates plugins by capability or identifier. Of several enabled plugins that
// declare a capability, the one with the lowest id is found.
type PluginResolver interface {
	// ResolveCapability returns the plugin for the capability. Where none answers, the error is an
	// *UnavailableError saying whether none is installed or every one is disabled.
	ResolveCapability(capability string) (DiscoveredPlugin, error)
	FindPluginByCapability(capability string) (DiscoveredPlugin, bool)
	GetPlugin(id string) (DiscoveredPlugin, bool)
}

// PluginActivator activates plugins and manages active references.
type PluginActivator interface {
	EnsureActive(ctx context.Context, pluginID string) (Endpoint, error)
	Release(pluginID string)
}

// PluginAcquirer increments the active reference count for a plugin.
type PluginAcquirer interface {
	Acquire(pluginID string)
}

// PluginAuthorizer verifies plugin permissions.
type PluginAuthorizer interface {
	CheckPluginPermission(pluginID, permission string) error
}

// PluginPathValidator validates paths within the vault boundary for plugins.
type PluginPathValidator interface {
	GetVaultPath() string
	ValidatePluginPath(pluginID, targetPath string) (string, error)
}

// PluginDiscoverer scans for available plugins and returns their metadata in the order of their ids.
type PluginDiscoverer interface {
	Discover() error
	GetPluginInfos() []PluginInfo
}

// PluginConfigurator manages configuration and settings for plugins.
type PluginConfigurator interface {
	SetSettings(all map[string]map[string]any)
	SetPersistHandler(handler PluginPersistHandler)
	NotifyAllPluginConfigs(ctx context.Context) error
	SetCompleter(completer Completer)
	SetKeyStore(store KeyStore)
	SetPluginSecret(ctx context.Context, pluginID, name, key string) error
	DeletePluginSecret(ctx context.Context, pluginID, name string) error
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
}

// PluginEventEmitter broadcasts events to plugins.
type PluginEventEmitter interface {
	EmitEvent(event string)
}

// PluginManager coordinates process lifecycle, discovery, settings, and RPC routing for modular extensions.
type PluginManager interface {
	PluginDiscoverer
	CompletionChoices
	PluginResolver
	PluginActivator
	PluginAcquirer
	PluginConfigurator
	PluginEventEmitter
	PluginAuthorizer
	PluginPathValidator
	// SetSafeMode stops every running plugin when enabled, and starts none until it is disabled.
	SetSafeMode(enabled bool)
	// SetVaultPath is the vault the plugins' paths are held inside.
	SetVaultPath(path string)
	// SetOnPluginFailed registers a callback invoked once when a plugin has failed to start repeatedly,
	// and again when a retry of it fails.
	SetOnPluginFailed(fn func(PluginFailure))
	// SetOnPluginRecovered registers a callback invoked with the id of a plugin the host stops holding
	// off.
	SetOnPluginRecovered(fn func(pluginID string))
	// RetryPlugin forgets the failures of a plugin and starts it once. The failure is set where the
	// plugin fails again; an unknown plugin is an *UnavailableError.
	RetryPlugin(ctx context.Context, pluginID string) (*PluginFailure, error)
	// TeardownAll stops every plugin, giving each the grace period to exit.
	TeardownAll(ctx context.Context, gracePeriod time.Duration) error
}
