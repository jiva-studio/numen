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
	takeConfigSnapshot func(pluginID string) (configSnapshot, bool, error)
	listPluginIDs      func() []string
	reportError        func(err error)
}

// secretDelivery gives the plugins their secrets: it keeps what each plugin was last given and
// sends Configure when that changes.
type secretDelivery struct {
	host                   secretHost
	lock                   chan struct{}
	mu                     sync.RWMutex
	keyStore               pluginport.KeyStore
	source                 pluginport.SecretSource
	delivered              map[string]string
	awaitingFirstConfigure map[string]bool
	configWatchers         map[string]map[int]func()
	nextWatcher            int
	noKeyStoreOnce         sync.Once
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

func (d *secretDelivery) unlockSecrets() { <-d.lock }

func (d *secretDelivery) startActivation(pluginID string) {
	d.mu.Lock()
	delete(d.delivered, pluginID)
	d.awaitingFirstConfigure[pluginID] = true
	d.mu.Unlock()
}

func (d *secretDelivery) endActivation(pluginID string) {
	d.mu.Lock()
	delete(d.awaitingFirstConfigure, pluginID)
	d.mu.Unlock()
}

func (d *secretDelivery) isAwaitingFirstConfigure(pluginID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.awaitingFirstConfigure[pluginID]
}

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

func (d *secretDelivery) refresh(ctx context.Context) error {
	if err := d.lockSecrets(ctx); err != nil {
		return err
	}
	defer d.unlockSecrets()
	ids := slices.DeleteFunc(d.host.listPluginIDs(), func(id string) bool { return !d.hasSecretsChanged(ctx, id) })
	return d.configurePlugins(ctx, ids)
}

func (d *secretDelivery) listSecretSignatures(ctx context.Context) map[string]string {
	ids := d.host.listPluginIDs()
	signatures := make(map[string]string, len(ids))
	for _, id := range ids {
		signatures[id] = d.getSecretsSignatureOf(ctx, id)
	}
	return signatures
}

func (d *secretDelivery) getSecretsSignatureOf(ctx context.Context, pluginID string) string {
	snapshot, isRunning, err := d.takeSnapshot(pluginID)
	if !isRunning || err != nil || d.isAwaitingFirstConfigure(pluginID) {
		return ""
	}
	secrets, _ := d.listConfigureSecrets(ctx, snapshot, false)
	return "list:" + getSecretsSignature(secrets)
}

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

func describeSecretValues(secrets []secret.Secret) []*v1.SecretValue {
	values := make([]*v1.SecretValue, 0, len(secrets))
	for _, one := range secrets {
		values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group})
	}
	return values
}

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

func describeConfigureFailure(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	return connect.CodeOf(err).String()
}
