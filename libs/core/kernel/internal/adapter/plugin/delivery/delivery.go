package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ConfigureTimeout bounds one Configure call to a plugin.
const ConfigureTimeout = 3 * time.Second

// Snapshot is what a Configure call to one plugin is made of, read while the table of plugins is locked.
type Snapshot struct {
	PluginID   string
	Endpoint   transport.Endpoint
	ConfigJSON string
	Values     map[string]string
	Declared   []manifest.SecretDeclaration
	Requested  []string
	store      pluginport.KeyStore
	source     plugindomain.SecretSource
}

// Host is what the delivery asks the plugin host for.
type Host interface {
	GetSnapshot(pluginID string) (Snapshot, bool, error)
	ListPluginIDs() []string
	ReportError(err error)
	GetClient(endpoint transport.Endpoint) *http.Client
}

// Delivery gives each running plugin its configuration and the secrets it declares, and sends them
// again when the list of secrets allowed to it changes.
type Delivery struct {
	host           Host
	lock           chan struct{}
	mu             sync.RWMutex
	keyStore       pluginport.KeyStore
	source         plugindomain.SecretSource
	ledger         *ledger
	watchers       *watchers
	noKeyStoreOnce sync.Once
}

// New creates a delivery over the host.
func New(host Host) *Delivery {
	return &Delivery{host: host, lock: make(chan struct{}, 1), ledger: newLedger(), watchers: &watchers{}}
}

// SetKeyStore sets where the values of secrets are kept.
func (d *Delivery) SetKeyStore(store pluginport.KeyStore) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.keyStore = store
}

// SetSource sets where the list of secrets is read from.
func (d *Delivery) SetSource(source plugindomain.SecretSource) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.source = source
}

// GetSource returns where the list of secrets is read from.
func (d *Delivery) GetSource() plugindomain.SecretSource {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.source
}

func (d *Delivery) getSnapshot(pluginID string) (Snapshot, bool, error) {
	snapshot, isRunning, err := d.host.GetSnapshot(pluginID)
	if !isRunning || err != nil {
		return snapshot, isRunning, err
	}
	d.mu.RLock()
	snapshot.store = d.keyStore
	snapshot.source = d.source
	d.mu.RUnlock()
	return snapshot, true, nil
}

// Lock holds the deliveries to every plugin to the caller until Unlock, or until ctx ends.
func (d *Delivery) Lock(ctx context.Context) error {
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

// Unlock releases the hold Lock took.
func (d *Delivery) Unlock() { <-d.lock }

// AwaitFirstConfigure marks the plugin as not yet configured since it started, which holds back the
// deliveries of Refresh to it.
func (d *Delivery) AwaitFirstConfigure(pluginID string) {
	d.ledger.markAwaitingFirstConfigure(pluginID)
}

// ClearFirstConfigure ends the wait AwaitFirstConfigure began.
func (d *Delivery) ClearFirstConfigure(pluginID string) {
	d.ledger.clearAwaitingFirstConfigure(pluginID)
}

// IsAwaitingFirstConfigure reports whether the wait AwaitFirstConfigure began is under way.
func (d *Delivery) IsAwaitingFirstConfigure(pluginID string) bool {
	return d.ledger.isAwaitingFirstConfigure(pluginID)
}

// WatchConfig registers fn to run each time the plugin has been given its configuration. The returned
// function removes the registration.
func (d *Delivery) WatchConfig(pluginID string, fn func()) (stop func()) {
	return d.watchers.add(pluginID, fn)
}

// Refresh sends Configure to each running plugin whose list of secrets differs from the last it was
// given, and to each plugin in forced.
func (d *Delivery) Refresh(ctx context.Context, forced []string) error {
	d.ledger.forget(forced)
	if err := d.Lock(ctx); err != nil {
		return err
	}
	defer d.Unlock()
	ids := d.host.ListPluginIDs()
	failures := make([]error, len(ids))
	var group sync.WaitGroup
	for i, id := range ids {
		if d.ledger.isAwaitingFirstConfigure(id) {
			continue
		}
		group.Go(func() {
			failures[i] = d.refreshPlugin(ctx, id)
		})
	}
	group.Wait()
	return errors.Join(failures...)
}

func (d *Delivery) refreshPlugin(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.getSnapshot(pluginID)
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}
	lists, listErr := listAllowed(snapshot)
	if listErr == nil && !d.ledger.isDue(pluginID, getSignature(lists)) {
		return nil
	}
	return d.deliver(ctx, snapshot, lists, listErr)
}

// Configure sends Configure to the plugin whether or not its secrets changed. The caller holds Lock.
func (d *Delivery) Configure(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.getSnapshot(pluginID)
	if !isRunning || err != nil {
		return err
	}
	lists, listErr := listAllowed(snapshot)
	return d.deliver(ctx, snapshot, lists, listErr)
}

func (d *Delivery) deliver(ctx context.Context, snapshot Snapshot, lists []allowedList, listErr error) error {
	entries, failures := readEntries(ctx, snapshot, lists)
	readErr := d.getReadError(listErr, failures)
	isConfigured, sendErr := d.sendConfigure(ctx, snapshot, entries)
	d.ledger.recordOutcome(snapshot.PluginID, getSignature(lists), sendErr, readErr, sendErr != nil && isCallerFailure(ctx, sendErr))
	if sendErr == nil && isConfigured {
		d.watchers.run(snapshot.PluginID)
	}
	return errors.Join(sendErr, readErr)
}

func isCallerFailure(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	code := connect.CodeOf(err)
	return code == connect.CodeCanceled || code == connect.CodeDeadlineExceeded
}

func (d *Delivery) sendConfigure(ctx context.Context, snapshot Snapshot, entries []*v1.ConfigureSecrets) (bool, error) {
	client := numenv1connect.NewPluginLifecycleServiceClient(
		d.host.GetClient(snapshot.Endpoint),
		transport.GetBaseURL(snapshot.Endpoint),
		connect.WithInterceptors(transport.AuthInterceptor(snapshot.Endpoint.Token)),
	)
	req := connect.NewRequest(&v1.ConfigureRequest{
		PluginId:          snapshot.PluginID,
		ConfigurationJson: snapshot.ConfigJSON,
		Values:            snapshot.Values,
		Secrets:           entries,
	})
	callCtx, cancel := context.WithTimeout(ctx, ConfigureTimeout)
	defer cancel()
	_, err := client.Configure(callCtx, req)
	if err == nil {
		return true, nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && len(snapshot.Declared) == 0 {
		return false, nil
	}
	return false, fmt.Errorf("configure plugin %s: %s", snapshot.PluginID, getConfigureFailureText(err))
}

func (d *Delivery) getReadError(listErr error, failures []error) error {
	problems := []error{listErr}
	for _, failure := range failures {
		if errors.Is(failure, plugindomain.ErrNoKeyStore) {
			d.noKeyStoreOnce.Do(func() { d.host.ReportError(failure) })
			continue
		}
		problems = append(problems, failure)
	}
	return errors.Join(problems...)
}

func getConfigureFailureText(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	return connect.CodeOf(err).String()
}
