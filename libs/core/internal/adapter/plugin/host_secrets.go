package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
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
	h.secretSource = source
	h.mu.Unlock()
	h.pruneSecretPermissions()
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
		declarations = appendDeclarations(declarations, p)
	}
	return declarations
}

// configureCallTimeout is how long the host waits for one Configure call.
const configureCallTimeout = 3 * time.Second

// ApplySecretChange runs change while no other secret operation runs. It then sends Configure to
// each running plugin whose list of secrets differs from the one it had before change, and to each
// plugin whose id change returns, and waits for every send. When a send fails the change stays
// committed and the error names the plugin. It returns the context error, with change not run, when
// ctx ends before the lock is free.
func (h *Host) ApplySecretChange(ctx context.Context, change func() (readers []string, err error)) error {
	if err := h.lockSecrets(ctx); err != nil {
		return err
	}
	defer h.unlockSecrets()
	before := h.listSecretSignatures(ctx)
	readers, err := change()
	if err != nil {
		return err
	}
	var ids []string
	for id, signature := range before {
		if slices.Contains(readers, id) || h.getSecretsSignatureOf(ctx, id) != signature {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return h.configurePlugins(ctx, ids)
}

// RefreshSecrets sends Configure to each running plugin whose list of secrets differs from the one
// it was last given.
func (h *Host) RefreshSecrets(ctx context.Context) error {
	if err := h.lockSecrets(ctx); err != nil {
		return err
	}
	defer h.unlockSecrets()
	h.mu.RLock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	h.mu.RUnlock()
	ids = slices.DeleteFunc(ids, func(id string) bool { return !h.hasSecretsChanged(ctx, id) })
	return h.configurePlugins(ctx, ids)
}

// listSecretSignatures returns the signature of the list of secrets of every plugin. The caller
// holds secretLock.
func (h *Host) listSecretSignatures(ctx context.Context) map[string]string {
	h.mu.RLock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	h.mu.RUnlock()
	signatures := make(map[string]string, len(ids))
	for _, id := range ids {
		signatures[id] = h.getSecretsSignatureOf(ctx, id)
	}
	return signatures
}

// getSecretsSignatureOf returns the signature of the list of secrets the plugin would be given now.
// A plugin that is not running, or has not had its activation Configure, has the empty signature.
func (h *Host) getSecretsSignatureOf(ctx context.Context, pluginID string) string {
	snapshot, isRunning, err := h.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil || h.isAwaitingFirstConfigure(pluginID) {
		return ""
	}
	secrets, _ := h.listConfigureSecrets(ctx, snapshot, false)
	return "list:" + getSecretsSignature(secrets)
}

// isAwaitingFirstConfigure reports whether the plugin is between its activation and the Configure
// of that activation.
func (h *Host) isAwaitingFirstConfigure(pluginID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.awaitingFirstConfigure[pluginID]
}

// configurePlugins sends Configure to the plugins in turn and returns the failures joined. A plugin
// that awaits the Configure of its activation gets none, because that one reads the list as it is.
// The caller holds secretLock.
func (h *Host) configurePlugins(ctx context.Context, ids []string) error {
	var failures []error
	for _, id := range ids {
		if h.isAwaitingFirstConfigure(id) {
			continue
		}
		readErr, configureErr := h.configurePluginLocked(ctx, id)
		failures = append(failures, configureErr, readErr)
	}
	return errors.Join(failures...)
}

// configurePluginLocked sends the configuration to the plugin and returns the failure to read its
// secrets apart from the failure of the Configure call. A plugin that is not running gets no call.
// The caller holds secretLock.
func (h *Host) configurePluginLocked(ctx context.Context, pluginID string) (readErr, configureErr error) {
	snapshot, isRunning, err := h.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		return err, nil
	}
	secrets, failures := h.listConfigureSecrets(ctx, snapshot, true)
	secretsErr := h.getSecretsError(failures)

	httpClient := transport.EndpointClient(snapshot.endpoint)
	client := numenv1connect.NewPluginLifecycleServiceClient(
		httpClient,
		transport.GetBaseURL(snapshot.endpoint),
		connect.WithInterceptors(transport.AuthInterceptor(snapshot.endpoint.Token)),
	)
	req := connect.NewRequest(&v1.ConfigureRequest{
		PluginId:          pluginID,
		ConfigurationJson: snapshot.configJSON,
		Values:            snapshot.values,
		Secrets:           secrets,
	})
	listed, _ := h.listConfigureSecrets(ctx, snapshot, false)
	signature := getSecretsSignature(listed)
	configCtx, cancel := context.WithTimeout(ctx, configureCallTimeout)
	defer cancel()
	_, err = client.Configure(configCtx, req)
	// A list that failed to arrive is remembered too, so a plain refresh does not send it again.
	h.mu.Lock()
	h.delivered[pluginID] = signature
	h.mu.Unlock()
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(snapshot.declared) == 0 {
			return secretsErr, nil
		}
		return secretsErr, fmt.Errorf("configure plugin %s: %s", pluginID, describeConfigureFailure(err))
	}
	h.runConfigWatchers(pluginID)
	return secretsErr, nil
}

// hasSecretsChanged reports whether the list of secrets of a running plugin differs from the one
// it was last given.
func (h *Host) hasSecretsChanged(ctx context.Context, pluginID string) bool {
	snapshot, isRunning, err := h.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		return false
	}
	secrets, _ := h.listConfigureSecrets(ctx, snapshot, false)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.awaitingFirstConfigure[pluginID] {
		return false
	}
	last, wasDelivered := h.delivered[pluginID]
	return !wasDelivered || last != getSecretsSignature(secrets)
}

// pruneSecretPermissions drops the permissions of declarations that are gone. It runs when the
// scan and the secret source are both there, in whichever order they come. A scan that had
// problems may have hidden an installed plugin, so it drops nothing.
func (h *Host) pruneSecretPermissions() {
	h.mu.RLock()
	prune := h.secretSource.Prune
	skip := !h.hasScanned || len(h.problems) > 0
	declared := h.listInstalledDeclarationsLocked()
	h.mu.RUnlock()
	if prune == nil || skip {
		return
	}
	if err := prune(declared); err != nil {
		h.reportError(fmt.Errorf("drop the permissions of removed declarations: %w", err))
	}
}

// listConfigureSecrets returns, for each active declaration of the plugin, the secrets it may read
// in the order it receives them. With withValues, each carries its value, a secret with no value on
// this machine is left out, and a declaration whose value cannot be read is not carried, so the
// plugin keeps what it has. Without it the key store is not asked and every allowed secret is
// listed, which is what the signature of the list is made of.
func (h *Host) listConfigureSecrets(ctx context.Context, snapshot configSnapshot, withValues bool) ([]*v1.ConfigureSecrets, []error) {
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
		if !withValues {
			entries = append(entries, &v1.ConfigureSecrets{Name: declaration.Name, Group: declaration.Group, Values: describeSecretValues(allowed)})
			continue
		}
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

// describeSecretValues lists the secrets without any value.
func describeSecretValues(secrets []secret.Secret) []*v1.SecretValue {
	values := make([]*v1.SecretValue, 0, len(secrets))
	for _, one := range secrets {
		values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group})
	}
	return values
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

// listInstalledDeclarationsLocked returns the declarations of every plugin installed in the plugin
// folders whatever its target, and of the registered plugins that are not on disk. A host of one
// target must not drop the permissions of a plugin that targets another window.
func (h *Host) listInstalledDeclarationsLocked() []pluginport.SecretPermission {
	var declarations []pluginport.SecretPermission
	installed := make(map[string]bool, len(h.installed))
	for _, p := range h.installed {
		installed[p.Manifest.ID] = true
		declarations = appendDeclarations(declarations, p)
	}
	for _, p := range h.getPluginsInOrderLocked() {
		if !installed[p.Manifest.ID] {
			declarations = appendDeclarations(declarations, p)
		}
	}
	return declarations
}

// appendDeclarations adds the secret declarations of the plugin to the list.
func appendDeclarations(declarations []pluginport.SecretPermission, p manifest.Discovered) []pluginport.SecretPermission {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, pluginport.SecretPermission{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}

// lockSecrets waits for the lock of the secret operations until ctx ends.
func (h *Host) lockSecrets(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case h.secretLock <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlockSecrets releases the lock taken by lockSecrets.
func (h *Host) unlockSecrets() { <-h.secretLock }

// describeConfigureFailure names the status of a failed Configure call and never the text of the plugin.
func describeConfigureFailure(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	return connect.CodeOf(err).String()
}
