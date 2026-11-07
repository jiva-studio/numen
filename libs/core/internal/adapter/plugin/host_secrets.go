package plugin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
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

// configureState is what the host keeps for the Configure calls of one plugin.
type configureState struct {
	// sendMu orders the Configure calls to the plugin, so that the later one carries the later list.
	sendMu sync.Mutex
	// cancel ends the call that is in flight. Host.mu guards it and the fields below.
	cancel context.CancelFunc
	// isQueued is set while a delivery is waiting for the one in flight, and isDelivering while a
	// goroutine runs the queued deliveries.
	isQueued, isDelivering bool
	// isForced is set when a queued delivery is owed even to a plugin whose list did not change.
	isForced bool
	// hasFailed is set while the plugin is marked as failed because of a Configure call.
	hasFailed bool
}

// getConfigureState returns the state of the plugin's Configure calls, creating it.
func (h *Host) getConfigureState(pluginID string) *configureState {
	h.mu.Lock()
	defer h.mu.Unlock()
	state := h.configure[pluginID]
	if state == nil {
		state = &configureState{}
		h.configure[pluginID] = state
	}
	return state
}

// ApplySecretChange runs change while no other secret operation runs, and then configures the
// running plugins whose list of secrets changed, and those whose ids change returns. The Configure
// calls run after it returns and do not delay or fail it.
func (h *Host) ApplySecretChange(ctx context.Context, change func() (readers []string, err error)) error {
	h.secretMu.Lock()
	readers, err := change()
	if err != nil {
		h.secretMu.Unlock()
		return err
	}
	ids := h.listPluginsToConfigure(ctx, readers)
	h.secretMu.Unlock()
	h.queueConfigure(context.WithoutCancel(ctx), ids, readers)
	return nil
}

// RefreshSecrets configures the running plugins whose list of secrets differs from the one they
// were last given.
func (h *Host) RefreshSecrets(ctx context.Context) error {
	h.secretMu.Lock()
	ids := h.listPluginsToConfigure(ctx, nil)
	h.secretMu.Unlock()

	var failures []error
	for _, id := range ids {
		readErr, configureErr := h.configurePlugin(ctx, id)
		h.recordConfigureResult(id, configureErr)
		if readErr != nil {
			failures = append(failures, readErr)
		}
	}
	return errors.Join(failures...)
}

// listPluginsToConfigure returns the ids of the plugins named in forced and of every running plugin
// whose list of secrets differs from the one it was last given. The caller holds secretMu.
func (h *Host) listPluginsToConfigure(ctx context.Context, forced []string) []string {
	h.mu.RLock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	h.mu.RUnlock()
	return slices.DeleteFunc(ids, func(id string) bool {
		return !slices.Contains(forced, id) && !h.hasSecretsChanged(ctx, id)
	})
}

// queueConfigure configures the plugins on goroutines of their own. A plugin is configured by one
// goroutine at a time, and deliveries queued while it works are merged into one that reads the list
// when it runs, so that the later list is the one the plugin ends with.
func (h *Host) queueConfigure(ctx context.Context, ids, forced []string) {
	for _, id := range ids {
		state := h.getConfigureState(id)
		h.mu.Lock()
		state.isQueued = true
		state.isForced = state.isForced || slices.Contains(forced, id)
		start := !state.isDelivering
		state.isDelivering = true
		h.mu.Unlock()
		if start {
			go h.runQueuedConfigures(ctx, id, state)
		}
	}
}

// runQueuedConfigures configures the plugin until no delivery is queued.
func (h *Host) runQueuedConfigures(ctx context.Context, pluginID string, state *configureState) {
	for {
		h.mu.Lock()
		if !state.isQueued {
			state.isDelivering = false
			h.mu.Unlock()
			return
		}
		state.isQueued = false
		isForced := state.isForced
		state.isForced = false
		h.mu.Unlock()
		if !isForced && !h.hasSecretsChangedUnderLock(ctx, pluginID) {
			continue
		}
		readErr, configureErr := h.configurePlugin(ctx, pluginID)
		if readErr != nil {
			h.reportError(readErr)
		}
		h.recordConfigureResult(pluginID, configureErr)
	}
}

// recordConfigureResult marks the plugin as failed when its Configure call failed, so that the
// plugin page shows it, and clears the mark when a later call succeeded.
func (h *Host) recordConfigureResult(pluginID string, configureErr error) {
	state := h.getConfigureState(pluginID)
	h.mu.Lock()
	hadFailed := state.hasFailed
	state.hasFailed = configureErr != nil
	sup := h.supervisors[pluginID]
	h.mu.Unlock()
	switch {
	case configureErr != nil:
		if sup != nil && !sup.IsActivationFailed() {
			h.recordActivationFailure(pluginID, configureErr)
			h.deliverPluginFailures()
		}
	case hadFailed && sup != nil:
		h.clearFailure(pluginID, sup)
	}
}

// configurePlugin sends the configuration to the plugin and returns the failure to read its secrets
// apart from the failure of the Configure call. The list of secrets is read when the call is about to
// be sent. A call is not sent to a plugin that is not running, and its outcome is dropped when the
// process it went to ended.
func (h *Host) configurePlugin(ctx context.Context, pluginID string) (readErr, configureErr error) {
	state := h.getConfigureState(pluginID)
	state.sendMu.Lock()
	defer state.sendMu.Unlock()

	h.secretMu.Lock()
	snapshot, isRunning, err := h.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		h.secretMu.Unlock()
		return err, nil
	}
	secrets, failures := h.listConfigureSecrets(ctx, snapshot)
	h.secretMu.Unlock()
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
	configCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	h.mu.Lock()
	state.cancel = cancel
	h.mu.Unlock()
	_, err = client.Configure(configCtx, req)

	h.mu.Lock()
	state.cancel = nil
	isCurrent := h.endpoints[pluginID].Token == snapshot.endpoint.Token
	h.mu.Unlock()
	if !isCurrent {
		return secretsErr, nil
	}
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(snapshot.declared) == 0 {
			return secretsErr, nil
		}
		return secretsErr, fmt.Errorf("configure plugin %s: %w", pluginID, err)
	}
	h.mu.Lock()
	h.delivered[pluginID] = getSecretsSignature(secrets)
	h.mu.Unlock()
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
	secrets, _ := h.listConfigureSecrets(ctx, snapshot)
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

// hasSecretsChangedUnderLock is hasSecretsChanged for a caller that does not hold secretMu.
func (h *Host) hasSecretsChangedUnderLock(ctx context.Context, pluginID string) bool {
	h.secretMu.Lock()
	defer h.secretMu.Unlock()
	return h.hasSecretsChanged(ctx, pluginID)
}
