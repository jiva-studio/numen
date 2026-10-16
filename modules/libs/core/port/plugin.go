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

// Endpoint holds connection parameters and credentials for an active plugin.
type Endpoint struct {
	Host      string
	Port      int
	Token     string
	Transport any
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

// PluginManager coordinates process lifecycle, discovery, settings, and RPC routing for modular extensions.
type PluginManager interface {
	Discover() error
	GetPluginInfos() []PluginInfo
	FindPluginByCapability(capability string) (DiscoveredPlugin, bool)
	FindPluginByFileExtension(ext string) (DiscoveredPlugin, bool)
	GetPlugin(id string) (DiscoveredPlugin, bool)
	EnsureActive(ctx context.Context, pluginID string) (Endpoint, error)
	Acquire(pluginID string)
	Release(pluginID string)
	SetSettings(all map[string]map[string]any)
	SetPersistHandler(handler PluginPersistHandler)
	NotifyAllPluginConfigs(ctx context.Context) error
	UpdatePluginSetting(ctx context.Context, pluginID, key string, value any) error
	EmitEvent(event string)
	GetVaultPath() string
	CheckPluginPermission(pluginID, permission string) error
	ValidatePluginPath(pluginID, targetPath string) (string, error)
}
