package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// configureCallTimeout is how long the host waits for one Configure call.
const configureCallTimeout = 3 * time.Second

// secretHost is what secretDelivery reads from the host that owns it.
type secretHost struct {
	getConfigSnapshot func(pluginID string) (configSnapshot, bool, error)
	listPluginIDs     func() []string
	reportError       func(err error)
}

// secretDelivery gives the plugins their secrets: it keeps what each plugin was last given and
// sends Configure when that changes. The lock makes the rounds of sends one at a time and is held
// for the sends only, never while a secret is written.
//
// A list counts as held only after a send that succeeded with every value read. After any other
// outcome the plugin is forgotten and the next refresh sends again. The one exception is a plugin
// that answered with an error of its own: that list is kept in rejected, and a refresh does not send
// it again until the list changes, an operation names the plugin or the process restarts. A send cut
// short by the context of the caller, a cancel or a timeout, says nothing about the plugin and is
// never kept as rejected.
type secretDelivery struct {
	host                   secretHost
	lock                   chan struct{}
	mu                     sync.RWMutex
	keyStore               pluginport.KeyStore
	source                 pluginport.SecretSource
	delivered              map[string]string
	rejected               map[string]string
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
		rejected:               make(map[string]string),
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

func (d *secretDelivery) getSnapshot(pluginID string) (configSnapshot, bool, error) {
	snapshot, isRunning, err := d.host.getConfigSnapshot(pluginID)
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

func (d *secretDelivery) markAwaitingFirstConfigure(pluginID string) {
	d.mu.Lock()
	delete(d.delivered, pluginID)
	delete(d.rejected, pluginID)
	d.awaitingFirstConfigure[pluginID] = true
	d.mu.Unlock()
}

func (d *secretDelivery) clearAwaitingFirstConfigure(pluginID string) {
	d.mu.Lock()
	delete(d.awaitingFirstConfigure, pluginID)
	d.mu.Unlock()
}

func (d *secretDelivery) isAwaitingFirstConfigure(pluginID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.awaitingFirstConfigure[pluginID]
}

func (d *secretDelivery) forget(pluginIDs []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, id := range pluginIDs {
		delete(d.delivered, id)
		delete(d.rejected, id)
	}
}

func (d *secretDelivery) refresh(ctx context.Context, forced []string) error {
	d.forget(forced)
	if err := d.lockSecrets(ctx); err != nil {
		return err
	}
	defer d.unlockSecrets()
	ids := d.host.listPluginIDs()
	failures := make([]error, len(ids))
	var group sync.WaitGroup
	for i, id := range ids {
		if d.isAwaitingFirstConfigure(id) {
			continue
		}
		group.Add(1)
		go func() {
			defer group.Done()
			failures[i] = d.refreshPlugin(ctx, id)
		}()
	}
	group.Wait()
	return errors.Join(failures...)
}

func (d *secretDelivery) refreshPlugin(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.getSnapshot(pluginID)
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}
	lists, listErr := d.listAllowed(snapshot)
	if listErr == nil && !d.isDue(pluginID, getSignature(lists)) {
		return nil
	}
	return d.deliver(ctx, snapshot, lists, listErr)
}

func (d *secretDelivery) configurePlugin(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.getSnapshot(pluginID)
	if !isRunning || err != nil {
		return err
	}
	lists, listErr := d.listAllowed(snapshot)
	return d.deliver(ctx, snapshot, lists, listErr)
}

func (d *secretDelivery) isDue(pluginID, signature string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.awaitingFirstConfigure[pluginID] {
		return false
	}
	if last, isDelivered := d.delivered[pluginID]; isDelivered && last == signature {
		return false
	}
	last, isRejected := d.rejected[pluginID]
	return !isRejected || last != signature
}

// allowedList is one declaration of a plugin with the secrets allowed to it, in the order they are sent.
type allowedList struct {
	declaration manifest.Secret
	secrets     []secret.Secret
}

func (d *secretDelivery) listAllowed(snapshot configSnapshot) ([]allowedList, error) {
	if len(snapshot.declared) == 0 {
		return nil, nil
	}
	var list []secret.Secret
	var listErr error
	if snapshot.source.List != nil {
		if list, listErr = snapshot.source.List(); listErr != nil {
			list = nil
			listErr = fmt.Errorf("read the secrets of plugin %s: %w", snapshot.pluginID, listErr)
		}
	}
	page := secret.ListInPageOrder(list, snapshot.requested)
	lists := make([]allowedList, 0, len(snapshot.declared))
	for _, declaration := range snapshot.declared {
		lists = append(lists, allowedList{
			declaration: declaration,
			secrets: secret.ListAllowed(page, secret.Permission{
				Plugin: snapshot.pluginID, Name: declaration.Name, Group: declaration.Group,
			}),
		})
	}
	return lists, listErr
}

func readEntries(ctx context.Context, snapshot configSnapshot, lists []allowedList) ([]*v1.ConfigureSecrets, []error) {
	entries := make([]*v1.ConfigureSecrets, 0, len(lists))
	var failures []error
	for _, list := range lists {
		values, err := readSecretValues(ctx, snapshot.store, list.secrets)
		if err != nil {
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", list.declaration.Name, snapshot.pluginID, err))
		}
		entries = append(entries, &v1.ConfigureSecrets{Name: list.declaration.Name, Group: list.declaration.Group, Values: values})
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
			return values, err
		}
	}
	return values, nil
}

func (d *secretDelivery) deliver(ctx context.Context, snapshot configSnapshot, lists []allowedList, listErr error) error {
	entries, failures := readEntries(ctx, snapshot, lists)
	readErr := d.getReadError(listErr, failures)
	isConfigured, sendErr := d.sendConfigure(ctx, snapshot, entries)
	d.recordOutcome(ctx, snapshot.pluginID, getSignature(lists), sendErr, readErr)
	if sendErr == nil && isConfigured {
		d.runConfigWatchers(snapshot.pluginID)
	}
	return errors.Join(sendErr, readErr)
}

func (d *secretDelivery) recordOutcome(ctx context.Context, pluginID, signature string, sendErr, readErr error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case sendErr == nil && readErr == nil:
		d.delivered[pluginID] = signature
		delete(d.rejected, pluginID)
	case sendErr == nil:
		delete(d.delivered, pluginID)
		delete(d.rejected, pluginID)
	case isCallerFailure(ctx, sendErr):
		delete(d.delivered, pluginID)
	default:
		delete(d.delivered, pluginID)
		d.rejected[pluginID] = signature
	}
}

func isCallerFailure(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	code := connect.CodeOf(err)
	return code == connect.CodeCanceled || code == connect.CodeDeadlineExceeded
}

func (d *secretDelivery) sendConfigure(ctx context.Context, snapshot configSnapshot, entries []*v1.ConfigureSecrets) (bool, error) {
	client := numenv1connect.NewPluginLifecycleServiceClient(
		transport.EndpointClient(snapshot.endpoint),
		transport.GetBaseURL(snapshot.endpoint),
		connect.WithInterceptors(transport.AuthInterceptor(snapshot.endpoint.Token)),
	)
	req := connect.NewRequest(&v1.ConfigureRequest{
		PluginId:          snapshot.pluginID,
		ConfigurationJson: snapshot.configJSON,
		Values:            snapshot.values,
		Secrets:           entries,
	})
	callCtx, cancel := context.WithTimeout(ctx, configureCallTimeout)
	defer cancel()
	_, err := client.Configure(callCtx, req)
	if err == nil {
		return true, nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(snapshot.declared) == 0 {
		return false, nil
	}
	return false, fmt.Errorf("configure plugin %s: %s", snapshot.pluginID, getConfigureFailureText(err))
}

func getSignature(lists []allowedList) string {
	var signature strings.Builder
	for _, list := range lists {
		fmt.Fprintf(&signature, "%q/%q:", list.declaration.Name, list.declaration.Group)
		for _, one := range list.secrets {
			fmt.Fprintf(&signature, "%q/%q/%q,", one.ID, one.Name, one.Group)
		}
		signature.WriteString(";")
	}
	return signature.String()
}

func (d *secretDelivery) getReadError(listErr error, failures []error) error {
	problems := []error{listErr}
	for _, failure := range failures {
		if errors.Is(failure, pluginport.ErrNoKeyStore) {
			d.noKeyStoreOnce.Do(func() { d.host.reportError(failure) })
			continue
		}
		problems = append(problems, failure)
	}
	return errors.Join(problems...)
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

func getConfigureFailureText(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	return connect.CodeOf(err).String()
}
