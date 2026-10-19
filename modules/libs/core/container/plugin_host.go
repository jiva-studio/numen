package container

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// pluginGracePeriod is how long a plugin has to exit once it is told to stop.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

// resolvePlugin is the plugin host where an enabled plugin declares the capability. Where none
// does, the error is a *port.UnavailableError: an installation with no plugin host has none
// installed, and one in safe mode has every one disabled.
func (c Config) resolvePlugin(capability string) (port.PluginManager, error) {
	if c.PluginHost == nil {
		return nil, &port.UnavailableError{Capability: capability, Reason: port.PluginNotInstalled}
	}
	if c.SafeMode {
		return nil, &port.UnavailableError{Capability: capability, Reason: port.PluginDisabled, Err: errSafeMode}
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

// SetSafeMode is the configuration started in safe mode or out of it. In safe mode the plugin host
// stops every plugin it runs and starts none.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = enabled
	if enabled && c.PluginHost != nil {
		c.PluginHost.SetSafeMode(true)
	}
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are held inside.
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
