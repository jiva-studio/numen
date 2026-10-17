package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var errUndeclaredSecret = errors.New("the plugin declares no such secret")

// SetKeyStore sets where the secrets of plugins are kept.
func (h *Host) SetKeyStore(store port.KeyStore) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.keyStore = store
}

// SetPluginSecret stores the key of a secret the plugin declares and hands the
// plugin the secrets it holds now.
func (h *Host) SetPluginSecret(ctx context.Context, pluginID, name, key string) error {
	store, err := h.getSecretStore(pluginID, name)
	if err != nil {
		return err
	}
	if err := store.SetKey(ctx, getSecretName(pluginID, name), key); err != nil {
		return fmt.Errorf("store secret %s of plugin %s: %w", name, pluginID, err)
	}
	return h.NotifyPluginConfig(ctx, pluginID)
}

// DeletePluginSecret removes the key of a secret the plugin declares and hands
// the plugin the secrets it holds now.
func (h *Host) DeletePluginSecret(ctx context.Context, pluginID, name string) error {
	store, err := h.getSecretStore(pluginID, name)
	if err != nil {
		return err
	}
	if err := store.DeleteKey(ctx, getSecretName(pluginID, name)); err != nil {
		return fmt.Errorf("delete secret %s of plugin %s: %w", name, pluginID, err)
	}
	return h.NotifyPluginConfig(ctx, pluginID)
}

// getSecretStore returns the key store once the plugin is known to declare the
// secret.
func (h *Host) getSecretStore(pluginID, name string) (port.KeyStore, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil, fmt.Errorf("secret %s: %w", name, errPluginNotFound)
	}
	if !slices.Contains(p.Manifest.Secrets, name) {
		return nil, fmt.Errorf("plugin %s, secret %q: %w", pluginID, name, errUndeclaredSecret)
	}
	if h.keyStore == nil {
		return nil, fmt.Errorf("secret %s of plugin %s: %w", name, pluginID, port.ErrNoKeyStore)
	}
	return h.keyStore, nil
}

// getSecretName is the name a secret of a plugin is stored under.
func getSecretName(pluginID, name string) string {
	return pluginID + "/" + name
}

// readPluginSecrets reads the keys of the secrets a plugin declares. A secret
// without a key is empty, which clears the key the plugin holds; one the store
// fails to read is left out, so the plugin keeps what it has, and the failure is
// returned beside the secrets that were read.
func readPluginSecrets(
	ctx context.Context, store port.KeyStore, pluginID string, names []string,
) (map[string]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	if store == nil {
		return nil, fmt.Errorf("plugin %s declares secrets: %w", pluginID, port.ErrNoKeyStore)
	}
	secrets := make(map[string]string, len(names))
	var failures []error
	for _, name := range names {
		key, err := store.GetKey(ctx, getSecretName(pluginID, name))
		switch {
		case err == nil:
			secrets[name] = key
		case errors.Is(err, port.ErrNoKey):
			// An empty value clears the key the plugin holds.
			secrets[name] = ""
		default:
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", name, pluginID, err))
		}
	}
	return secrets, errors.Join(failures...)
}
