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
)

// configureCallTimeout is how long the host waits for one Configure call.
const configureCallTimeout = 3 * time.Second

// secretHost is what secretDelivery reads from the host that owns it.
type secretHost struct {
	// takeConfigSnapshot reads what a Configure call to the plugin is made of, and reports whether
	// the plugin runs. It leaves the key store and the secret source empty.
	takeConfigSnapshot func(pluginID string) (configSnapshot, bool, error)
	// listPluginIDs returns the ids of the registered plugins in order.
	listPluginIDs func() []string
	reportError   func(err error)
}

// secretDelivery gives the plugins their secrets: it keeps what each plugin was last given and
// sends Configure when that changes.
type secretDelivery struct {
	host secretHost
	// lock orders every operation on the secrets of every plugin and the Configure calls that
	// deliver them. It is locked before mu, never while mu is locked.
	lock chan struct{}
	mu   sync.RWMutex
	// keyStore and source are set by the app and read by every Configure call.
	keyStore pluginport.KeyStore
	source   pluginport.SecretSource
	// delivered is the list of secrets each plugin was last configured with, as a signature.
	delivered map[string]string
	// awaitingFirstConfigure holds the plugins that are Active and have not yet been through the Configure of their activation.
	awaitingFirstConfigure map[string]bool
	configWatchers         map[string]map[int]func()
	nextWatcher            int
	// noKeyStoreOnce reports once that a secret cannot be read on a machine without a key store.
	noKeyStoreOnce sync.Once
}

func newSecretDelivery(host secretHost) *secretDelivery {
	return &secretDelivery{
		host:                   host,
		lock:                   make(chan struct{}, 1),
		delivered:              make(map[string]string),
		awaitingFirstConfigure: make(map[string]bool),
	}
}

func (d *secretDelivery) setKeyStore(store pluginport.KeyStore) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.keyStore = store
}

func (d *secretDelivery) setSource(source pluginport.SecretSource) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.source = source
}

func (d *secretDelivery) getSource() pluginport.SecretSource {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.source
}

// takeSnapshot reads what a Configure call to the plugin is made of, and reports whether the plugin
// runs.
func (d *secretDelivery) takeSnapshot(pluginID string) (configSnapshot, bool, error) {
	snapshot, isRunning, err := d.host.takeConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		return snapshot, isRunning, err
	}
	d.mu.RLock()
	snapshot.store = d.keyStore
	snapshot.source = d.source
	d.mu.RUnlock()
	return snapshot, true, nil
}

// lockSecrets waits for the lock of the secret operations until ctx ends.
func (d *secretDelivery) lockSecrets(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case d.lock <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlockSecrets releases the lock taken by lockSecrets.
func (d *secretDelivery) unlockSecrets() { <-d.lock }

// startActivation marks the plugin as awaiting the Configure of its activation and forgets what it
// was given before.
func (d *secretDelivery) startActivation(pluginID string) {
	d.mu.Lock()
	delete(d.delivered, pluginID)
	d.awaitingFirstConfigure[pluginID] = true
	d.mu.Unlock()
}

// endActivation marks the plugin as no longer awaiting the Configure of its activation.
func (d *secretDelivery) endActivation(pluginID string) {
	d.mu.Lock()
	delete(d.awaitingFirstConfigure, pluginID)
	d.mu.Unlock()
}

// isAwaitingFirstConfigure reports whether the plugin is between its activation and the Configure
// of that activation.
func (d *secretDelivery) isAwaitingFirstConfigure(pluginID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.awaitingFirstConfigure[pluginID]
}

// applyChange runs change while no other secret operation runs. It then sends Configure to each
// running plugin whose list of secrets differs from the one it had before change, and to each
// plugin whose id change returns, and waits for every send. When a send fails the change stays
// committed and the error names the plugin. It returns the context error, with change not run, when
// ctx ends before the lock is free.
func (d *secretDelivery) applyChange(ctx context.Context, change func() (readers []string, err error)) error {
	if err := d.lockSecrets(ctx); err != nil {
		return err
	}
	defer d.unlockSecrets()
	before := d.listSecretSignatures(ctx)
	readers, err := change()
	if err != nil {
		return err
	}
	var ids []string
	for id, signature := range before {
		if slices.Contains(readers, id) || d.getSecretsSignatureOf(ctx, id) != signature {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return d.configurePlugins(ctx, ids)
}

// refresh sends Configure to each running plugin whose list of secrets differs from the one it was
// last given.
func (d *secretDelivery) refresh(ctx context.Context) error {
	if err := d.lockSecrets(ctx); err != nil {
		return err
	}
	defer d.unlockSecrets()
	ids := slices.DeleteFunc(d.host.listPluginIDs(), func(id string) bool { return !d.hasSecretsChanged(ctx, id) })
	return d.configurePlugins(ctx, ids)
}

// listSecretSignatures returns the signature of the list of secrets of every plugin. The caller
// holds the lock.
func (d *secretDelivery) listSecretSignatures(ctx context.Context) map[string]string {
	ids := d.host.listPluginIDs()
	signatures := make(map[string]string, len(ids))
	for _, id := range ids {
		signatures[id] = d.getSecretsSignatureOf(ctx, id)
	}
	return signatures
}

// getSecretsSignatureOf returns the signature of the list of secrets the plugin would be given now.
// A plugin that is not running, or has not had its activation Configure, has the empty signature.
func (d *secretDelivery) getSecretsSignatureOf(ctx context.Context, pluginID string) string {
	snapshot, isRunning, err := d.takeSnapshot(pluginID)
	if !isRunning || err != nil || d.isAwaitingFirstConfigure(pluginID) {
		return ""
	}
	secrets, _ := d.listConfigureSecrets(ctx, snapshot, false)
	return "list:" + getSecretsSignature(secrets)
}

// configurePlugins sends Configure to the plugins at once and returns the failures joined in the
// order of the ids. A plugin that awaits the Configure of its activation gets none, because that
// one reads the list as it is. A plugin that does not answer holds up only itself, so the wait is
// the longest single call and not their sum. The caller holds the lock.
func (d *secretDelivery) configurePlugins(ctx context.Context, ids []string) error {
	failures := make([][2]error, len(ids))
	var group sync.WaitGroup
	for i, id := range ids {
		if d.isAwaitingFirstConfigure(id) {
			continue
		}
		group.Add(1)
		go func() {
			defer group.Done()
			readErr, configureErr := d.configurePlugin(ctx, id)
			failures[i] = [2]error{configureErr, readErr}
		}()
	}
	group.Wait()
	var joined []error
	for _, pair := range failures {
		joined = append(joined, pair[0], pair[1])
	}
	return errors.Join(joined...)
}

// configurePlugin sends the configuration to the plugin and returns the failure to read its
// secrets apart from the failure of the Configure call. A plugin that is not running gets no call.
// The caller holds the lock.
func (d *secretDelivery) configurePlugin(ctx context.Context, pluginID string) (readErr, configureErr error) {
	snapshot, isRunning, err := d.takeSnapshot(pluginID)
	if !isRunning || err != nil {
		return err, nil
	}
	secrets, failures := d.listConfigureSecrets(ctx, snapshot, true)
	secretsErr := d.getSecretsError(failures)

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
	listed, _ := d.listConfigureSecrets(ctx, snapshot, false)
	signature := getSecretsSignature(listed)
	configCtx, cancel := context.WithTimeout(ctx, configureCallTimeout)
	defer cancel()
	_, err = client.Configure(configCtx, req)
	// A list that failed to arrive is remembered too, so a plain refresh does not send it again.
	d.mu.Lock()
	d.delivered[pluginID] = signature
	d.mu.Unlock()
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(snapshot.declared) == 0 {
			return secretsErr, nil
		}
		return secretsErr, fmt.Errorf("configure plugin %s: %s", pluginID, describeConfigureFailure(err))
	}
	d.runConfigWatchers(pluginID)
	return secretsErr, nil
}

// hasSecretsChanged reports whether the list of secrets of a running plugin differs from the one
// it was last given.
func (d *secretDelivery) hasSecretsChanged(ctx context.Context, pluginID string) bool {
	snapshot, isRunning, err := d.takeSnapshot(pluginID)
	if !isRunning || err != nil {
		return false
	}
	secrets, _ := d.listConfigureSecrets(ctx, snapshot, false)
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.awaitingFirstConfigure[pluginID] {
		return false
	}
	last, wasDelivered := d.delivered[pluginID]
	return !wasDelivered || last != getSecretsSignature(secrets)
}

// listConfigureSecrets returns, for each active declaration of the plugin, the secrets it may read
// in the order it receives them. With withValues, each carries its value, a secret with no value on
// this machine is left out, and a declaration whose value cannot be read is not carried, so the
// plugin keeps what it has. Without it the key store is not asked and every allowed secret is
// listed, which is what the signature of the list is made of.
func (d *secretDelivery) listConfigureSecrets(ctx context.Context, snapshot configSnapshot, withValues bool) ([]*v1.ConfigureSecrets, []error) {
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
		value, err := store.GetKey(ctx, secret.GetEntryName(one.ID))
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
func (d *secretDelivery) getSecretsError(failures []error) error {
	var notified []error
	for _, failure := range failures {
		if errors.Is(failure, pluginport.ErrNoKeyStore) {
			d.noKeyStoreOnce.Do(func() { d.host.reportError(failure) })
			continue
		}
		notified = append(notified, failure)
	}
	return errors.Join(notified...)
}

// watchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin, which a caller holding the lock of the
// secrets could not do. The returned function removes the registration.
func (d *secretDelivery) watchPluginConfig(pluginID string, fn func()) (stop func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.configWatchers == nil {
		d.configWatchers = make(map[string]map[int]func())
	}
	if d.configWatchers[pluginID] == nil {
		d.configWatchers[pluginID] = make(map[int]func())
	}
	id := d.nextWatcher
	d.nextWatcher++
	d.configWatchers[pluginID][id] = fn
	return func() {
		d.mu.Lock()
		defer d.mu.Unlock()
		delete(d.configWatchers[pluginID], id)
	}
}

// runConfigWatchers runs the functions registered for the plugin, each on its own goroutine.
func (d *secretDelivery) runConfigWatchers(pluginID string) {
	d.mu.RLock()
	watchers := make([]func(), 0, len(d.configWatchers[pluginID]))
	for _, fn := range d.configWatchers[pluginID] {
		watchers = append(watchers, fn)
	}
	d.mu.RUnlock()
	for _, fn := range watchers {
		go fn()
	}
}

// describeConfigureFailure names the status of a failed Configure call and never the text of the plugin.
func describeConfigureFailure(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	return connect.CodeOf(err).String()
}
