package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

var errUndeclaredSecret = errors.New("the plugin declares no such secret")

// SetKeyStore sets where the secrets of plugins are kept.
func (h *Host) SetKeyStore(store pluginport.KeyStore) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.keyStore = store
}

// lockSecrets takes the lock that orders the changes of one plugin's secrets
// and the Configure calls that deliver them, and returns the call that gives it
// back.
func (h *Host) lockSecrets(pluginID string) func() {
	h.secretLocksMu.Lock()
	lock, found := h.secretLocks[pluginID]
	if !found {
		if h.secretLocks == nil {
			h.secretLocks = make(map[string]*sync.Mutex)
		}
		lock = &sync.Mutex{}
		h.secretLocks[pluginID] = lock
	}
	h.secretLocksMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

// SetPluginSecret stores the key of a secret the plugin declares, records that
// the secret is stored, and hands the plugin the secrets it holds now.
func (h *Host) SetPluginSecret(ctx context.Context, pluginID, name, key string) error {
	defer h.lockSecrets(pluginID)()
	store, err := h.getSecretStore(pluginID, name)
	if err != nil {
		return err
	}
	if err := store.SetKey(ctx, getSecretName(pluginID, name), key); err != nil {
		return fmt.Errorf("store secret %s of plugin %s: %w", name, pluginID, err)
	}
	if err := h.recordStoredSecret(pluginID, name, true); err != nil {
		return err
	}
	return h.notifyPluginConfigLocked(ctx, pluginID)
}

// DeletePluginSecret removes the key, if any, of a secret the plugin declares,
// clears the record that it is stored, and hands the plugin the secrets it
// holds now.
func (h *Host) DeletePluginSecret(ctx context.Context, pluginID, name string) error {
	defer h.lockSecrets(pluginID)()
	store, err := h.getSecretStore(pluginID, name)
	if err != nil {
		return err
	}
	err = store.DeleteKey(ctx, getSecretName(pluginID, name))
	if err != nil && !errors.Is(err, pluginport.ErrNoKey) {
		return fmt.Errorf("delete secret %s of plugin %s: %w", name, pluginID, err)
	}
	if err = h.recordStoredSecret(pluginID, name, false); err != nil {
		return err
	}
	return h.notifyPluginConfigLocked(ctx, pluginID)
}

// getSecretStore returns the key store once the plugin is known to declare the
// secret.
func (h *Host) getSecretStore(pluginID, name string) (pluginport.KeyStore, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[pluginID]
	if !ok {
		return nil, fmt.Errorf("secret %s: %w", name, errPluginNotFound)
	}
	if !slices.Contains(p.Manifest.GetSecretNames(), name) {
		return nil, fmt.Errorf("plugin %s, secret %q: %w", pluginID, name, errUndeclaredSecret)
	}
	if h.keyStore == nil {
		return nil, fmt.Errorf("secret %s of plugin %s: %w", name, pluginID, pluginport.ErrNoKeyStore)
	}
	return h.keyStore, nil
}

// getSecretName is the name a secret of a plugin is stored under.
func getSecretName(pluginID, name string) string {
	return pluginID + "/" + name
}

// recordStoredSecret persists and keeps whether the secret of the plugin has a
// key in the key store.
func (h *Host) recordStoredSecret(pluginID, name string, isStored bool) error {
	h.mu.RLock()
	stored := h.getStoredSecretsLocked(pluginID)
	persist := h.persistHandler
	h.mu.RUnlock()

	stored = slices.DeleteFunc(stored, func(target string) bool { return target == name })
	if isStored {
		stored = append(stored, name)
	}
	slices.Sort(stored)
	if persist != nil {
		if err := persist(pluginID, pluginport.SettingStoredSecrets, stored); err != nil {
			return fmt.Errorf("record secret %s of plugin %s: %w", name, pluginID, err)
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.settings[pluginID]; !ok {
		h.settings[pluginID] = make(map[string]any)
	}
	h.settings[pluginID][pluginport.SettingStoredSecrets] = stored
	return nil
}

// getStoredSecretsLocked returns the names of the secrets of the plugin that
// are recorded as stored, among those its manifest declares.
func (h *Host) getStoredSecretsLocked(pluginID string) []string {
	var recorded []string
	switch target := h.settings[pluginID][pluginport.SettingStoredSecrets].(type) {
	case []string:
		recorded = target
	case []any:
		for _, entry := range target {
			if name, ok := entry.(string); ok {
				recorded = append(recorded, name)
			}
		}
	}
	declared := h.plugins[pluginID].Manifest.GetSecretNames()
	var stored []string
	for _, name := range recorded {
		if slices.Contains(declared, name) && !slices.Contains(stored, name) {
			stored = append(stored, name)
		}
	}
	return stored
}

// readPluginSecrets reads the keys of the secrets a plugin declares. The store
// is read only for a secret recorded as stored; any other secret is empty,
// which clears the key the plugin holds. A secret the store fails to read is
// left out, so the plugin keeps what it has, and the failure is returned beside
// the secrets that were read.
func readPluginSecrets(
	ctx context.Context, store pluginport.KeyStore, pluginID string, declared, stored []string,
) (map[string]string, []error) {
	if len(declared) == 0 {
		return nil, nil
	}
	secrets := make(map[string]string, len(declared))
	var failures []error
	for _, name := range declared {
		if !slices.Contains(stored, name) {
			secrets[name] = ""
			continue
		}
		if store == nil {
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", name, pluginID, pluginport.ErrNoKeyStore))
			continue
		}
		key, err := store.GetKey(ctx, getSecretName(pluginID, name))
		switch {
		case err == nil:
			secrets[name] = key
		case errors.Is(err, pluginport.ErrNoKey):
			// An empty value clears the key the plugin holds.
			secrets[name] = ""
		default:
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", name, pluginID, err))
		}
	}
	return secrets, failures
}

// getSecretsError returns the failures to read secrets that the caller is told
// of. A machine without a key store is reported once through the error handler
// and is not returned.
func (h *Host) getSecretsError(failures []error) error {
	var notified []error
	for _, failure := range failures {
		if errors.Is(failure, pluginport.ErrNoKeyStore) {
			h.noKeyStoreOnce.Do(func() { h.reportError(failure) })
			continue
		}
		notified = append(notified, failure)
	}
	return errors.Join(notified...)
}
