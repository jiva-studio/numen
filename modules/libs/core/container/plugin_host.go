package container

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/plugintool"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

// pluginGracePeriod is how long a plugin has to exit once it is told to stop.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

// resolvePlugin is the plugin host where an enabled plugin declares the capability. Where none
// does, the error is a *port.UnavailableError: an installation with no plugin host has none
// installed, and one in safe mode has every one disabled.
func (c Config) resolvePlugin(capability string) (pluginport.PluginManager, error) {
	if c.PluginHost == nil {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginNotInstalled}
	}
	if c.SafeMode {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginDisabled, Err: errSafeMode}
	}
	if _, err := c.PluginHost.ResolveCapability(capability); err != nil {
		return nil, err
	}
	return c.PluginHost, nil
}

// hasPlugin is whether an enabled plugin declares the capability.
func (c Config) hasPlugin(capability string) bool {
	_, err := c.resolvePlugin(capability)
	return err == nil
}

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
// In safe mode the host stops every plugin it runs and starts none.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = enabled
	if c.PluginHost != nil {
		c.PluginHost.SetSafeMode(enabled)
	}
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func (c Config) SetPluginVaultPath(path string) {
	if c.PluginHost != nil {
		c.PluginHost.SetVaultPath(path)
	}
}

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	if c.PluginHost == nil {
		return nil
	}
	return c.PluginHost.TeardownAll(ctx, pluginGracePeriod)
}

// GetPluginTools is the use cases that offer an agent the tools the plugins serve. Where there is
// no plugin host they offer none.
func (c Config) GetPluginTools() plugintool.Tools {
	var tools pluginport.PluginTools
	if c.PluginHost != nil {
		tools = plugin.OpenToolsBridge(c.PluginHost)
	}
	return plugintool.NewTools(tools)
}

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
}

// GetPluginServices is the forwarding of the services plugins serve to the windows. Where there is
// no plugin host it serves none.
func (c Config) GetPluginServices() PluginServices {
	if c.PluginHost == nil {
		return noPluginServices{}
	}
	return plugin.OpenServiceProxy(c.PluginHost)
}

// noPluginServices is the forwarding of an installation with no plugin host.
type noPluginServices struct{}

func (noPluginServices) Serves(string) bool { return false }

func (noPluginServices) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "no plugin serves this service", http.StatusNotFound)
}
