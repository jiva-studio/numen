package port

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrPathTraversal is returned when a path escapes the active vault boundary.
	ErrPathTraversal = errors.New("path traversal outside active vault is prohibited")

	// ErrPermissionDenied is returned when a plugin lacks the required permission.
	ErrPermissionDenied = errors.New("plugin permission denied")
)

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

// PluginResolver locates plugins by capability, file extension, or identifier. Of several enabled
// plugins that match a capability or an extension, the one with the lowest id is found.
type PluginResolver interface {
	FindPluginByCapability(capability string) (DiscoveredPlugin, bool)
	FindPluginByFileExtension(ext string) (DiscoveredPlugin, bool)
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
}
