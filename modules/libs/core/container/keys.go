package container

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/keychain"

	"github.com/jiva-studio/numen/modules/libs/core/port/pluginport"
)

var errNoPluginHost = errors.New("this installation has no plugin host to keep a secret for")

// GetKeyStore is where the keys of services are kept: the one the installation
// names, else the system keychain. A machine without a keychain fails with
// port.ErrNoKeyStore and keeps no key anywhere else.
func (c Config) GetKeyStore() pluginport.KeyStore {
	if c.KeyStore != nil {
		return c.KeyStore
	}
	return keychain.Store{}
}

// GetPluginSecretWriter stores, as the window asks, the key of a secret a plugin
// declares in its manifest, and hands the plugin the secrets it holds now.
func (c Config) GetPluginSecretWriter() func(ctx context.Context, pluginID, name, key string) error {
	return func(ctx context.Context, pluginID, name, key string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.SetPluginSecret(ctx, pluginID, name, key)
	}
}

// GetPluginSecretDeleter removes, as the window asks, the key of a secret a
// plugin declares in its manifest, and hands the plugin the secrets it holds now.
func (c Config) GetPluginSecretDeleter() func(ctx context.Context, pluginID, name string) error {
	return func(ctx context.Context, pluginID, name string) error {
		if c.PluginHost == nil {
			return errNoPluginHost
		}
		return c.PluginHost.DeletePluginSecret(ctx, pluginID, name)
	}
}
