package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// SetKeyStore sets where the values of secrets are kept.
func (h *Host) SetKeyStore(store pluginport.KeyStore) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.keyStore = store
}

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (h *Host) SetSecretSource(source pluginport.SecretSource) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.secretSource = source
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (h *Host) ListDeclarations() []pluginport.SecretPermission {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.listDeclarationsLocked()
}

func (h *Host) listDeclarationsLocked() []pluginport.SecretPermission {
	var declarations []pluginport.SecretPermission
	for _, p := range h.getPluginsInOrderLocked() {
		for _, declaration := range p.Manifest.Secrets {
			declarations = append(declarations, pluginport.SecretPermission{
				Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
			})
		}
	}
	return declarations
}

// ApplySecretChange runs change while no other secret operation runs, and then configures the
// running plugins whose list of secrets changed, and those whose ids change returns.
func (h *Host) ApplySecretChange(ctx context.Context, change func() (readers []string, err error)) error {
	h.secretMu.Lock()
	defer h.secretMu.Unlock()
	readers, err := change()
	if err != nil {
		return err
	}
	return h.configureChangedLocked(ctx, readers)
}

// RefreshSecrets configures the running plugins whose list of secrets differs from the one they
// were last given.
func (h *Host) RefreshSecrets(ctx context.Context) error {
	h.secretMu.Lock()
	defer h.secretMu.Unlock()
	return h.configureChangedLocked(ctx, nil)
}

// configureChangedLocked configures the plugins named in forced and every running plugin whose
// list of secrets differs from the one it was last given.
func (h *Host) configureChangedLocked(ctx context.Context, forced []string) error {
	h.mu.RLock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	h.mu.RUnlock()

	var failures []error
	for _, id := range ids {
		if !slices.Contains(forced, id) && !h.hasSecretsChanged(ctx, id) {
			continue
		}
		if err := h.notifyPluginConfigLocked(ctx, id); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// hasSecretsChanged reports whether the list of secrets of a running plugin differs from the one
// it was last given.
func (h *Host) hasSecretsChanged(ctx context.Context, pluginID string) bool {
	snapshot, isRunning, err := h.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		return false
	}
	secrets, _ := h.listConfigureSecrets(ctx, snapshot)
	h.mu.RLock()
	defer h.mu.RUnlock()
	last, wasDelivered := h.delivered[pluginID]
	return !wasDelivered || last != getSecretsSignature(secrets)
}

// pruneSecretPermissions drops the permissions of declarations that are gone. A scan that had
// problems may have hidden an installed plugin, so it drops nothing.
func (h *Host) pruneSecretPermissions() {
	h.mu.RLock()
	prune := h.secretSource.Prune
	hasProblems := len(h.problems) > 0
	declared := h.listDeclarationsLocked()
	h.mu.RUnlock()
	if prune == nil || hasProblems {
		return
	}
	if err := prune(declared); err != nil {
		h.reportError(fmt.Errorf("drop the permissions of removed declarations: %w", err))
	}
}

// listConfigureSecrets returns, for each active declaration of the plugin, the secrets it may read
// with their values, in the order it receives them. A secret with no value on this machine is left
// out. A declaration whose value cannot be read is not carried, so the plugin keeps what it has.
func (h *Host) listConfigureSecrets(ctx context.Context, snapshot configSnapshot) ([]*v1.ConfigureSecrets, []error) {
	if len(snapshot.declared) == 0 {
		return nil, nil
	}
	var list []secret.Secret
	if snapshot.source.List != nil {
		var err error
		if list, err = snapshot.source.List(); err != nil {
			return nil, []error{fmt.Errorf("read the secrets of plugin %s: %w", snapshot.pluginID, err)}
		}
	}
	page := secret.ListInPageOrder(list, snapshot.requested)
	var entries []*v1.ConfigureSecrets
	var failures []error
	for _, declaration := range snapshot.declared {
		allowed := secret.ListAllowed(page, secret.Permission{
			Plugin: snapshot.pluginID, Name: declaration.Name, Group: declaration.Group,
		})
		values, err := readSecretValues(ctx, snapshot.store, allowed)
		if err != nil {
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", declaration.Name, snapshot.pluginID, err))
			if !errors.Is(err, pluginport.ErrNoKeyStore) {
				continue
			}
		}
		entries = append(entries, &v1.ConfigureSecrets{Name: declaration.Name, Group: declaration.Group, Values: values})
	}
	return entries, failures
}

// readSecretValues reads the value of each secret. A secret the store has no value for is left
// out, and so is every secret on a machine without a store.
func readSecretValues(ctx context.Context, store pluginport.KeyStore, secrets []secret.Secret) ([]*v1.SecretValue, error) {
	values := make([]*v1.SecretValue, 0, len(secrets))
	for _, one := range secrets {
		if store == nil {
			return values, pluginport.ErrNoKeyStore
		}
		value, err := store.GetKey(ctx, "secret/"+one.ID)
		switch {
		case err == nil:
			values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group, Value: value})
		case errors.Is(err, pluginport.ErrNoKey):
		default:
			return nil, err
		}
	}
	return values, nil
}

// getSecretsSignature describes the list of secrets of a Configure call without any value: the
// declarations carried and, for each, the id, name and group of the secrets in order.
func getSecretsSignature(entries []*v1.ConfigureSecrets) string {
	var signature strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&signature, "%q/%q:", entry.GetName(), entry.GetGroup())
		for _, value := range entry.GetValues() {
			fmt.Fprintf(&signature, "%q/%q/%q,", value.GetId(), value.GetName(), value.GetGroup())
		}
		signature.WriteString(";")
	}
	return signature.String()
}

// getSecretsError returns the secret read failures that are reported to the caller. A machine
// without a key store is reported once through the error handler and is not returned.
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

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin, which a caller holding the lock of the
// secrets could not do. The returned function removes the registration.
func (h *Host) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.configWatchers == nil {
		h.configWatchers = make(map[string]map[int]func())
	}
	if h.configWatchers[pluginID] == nil {
		h.configWatchers[pluginID] = make(map[int]func())
	}
	id := h.nextWatcher
	h.nextWatcher++
	h.configWatchers[pluginID][id] = fn
	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.configWatchers[pluginID], id)
	}
}

func (h *Host) runConfigWatchers(pluginID string) {
	h.mu.RLock()
	watchers := make([]func(), 0, len(h.configWatchers[pluginID]))
	for _, fn := range h.configWatchers[pluginID] {
		watchers = append(watchers, fn)
	}
	h.mu.RUnlock()
	for _, fn := range watchers {
		go fn()
	}
}
