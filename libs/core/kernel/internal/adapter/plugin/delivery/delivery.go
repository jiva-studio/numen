package delivery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// ConfigureTimeout bounds one Configure call to a plugin.
const ConfigureTimeout = 3 * time.Second

// Snapshot is what a Configure call to one plugin is made of, read while the table of plugins is locked.
type Snapshot struct {
	PluginID   string
	Endpoint   transport.Endpoint
	ConfigJSON string
	Values     map[string]string
}

// Host is what the delivery asks the plugin host for. RecordConfigureAnswer is given the answer of the
// plugin to each Configure call it answered: nil where it took the configuration, a *RefusedError where
// it did not. Equal revisions of GetPluginConfigRevision are one configuration.
type Host interface {
	GetPluginConfigRevision(pluginID string) string
	GetSnapshot(pluginID string) (Snapshot, bool, error)
	ListPluginIDs() []string
	GetClient(endpoint transport.Endpoint) *http.Client
	RecordConfigureAnswer(pluginID string, refused *RefusedError)
}

// RefusedError is a Configure call the plugin answered with an error. Text is the message of the plugin,
// or the code of the answer where it has none.
type RefusedError struct {
	PluginID string
	Text     string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("configure plugin %s: %s", e.PluginID, e.Text)
}

// Delivery gives each running plugin its configuration, and sends it again when the plugin is forced.
type Delivery struct {
	host     Host
	lock     chan struct{}
	ledger   *ledger
	watchers *watchers
}

func New(host Host) *Delivery {
	return &Delivery{host: host, lock: make(chan struct{}, 1), ledger: newLedger(), watchers: &watchers{}}
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

// WatchConfig registers fn to run each time the plugin has been given its configuration, and each time
// Configure or a forced Refresh finds it not running, which is a change of its configuration it has not
// been given. The first configuration a started plugin is given runs fn only where it differs from the one
// fn last ran for, so a read that fn makes and that starts the plugin does not run fn again. The returned
// function removes the registration.
func (d *Delivery) WatchConfig(pluginID string, fn func()) (stop func()) {
	return d.watchers.add(pluginID, fn)
}

// Refresh sends Configure to each running plugin that was not given its configuration yet, and to
// each plugin in forced. A forced plugin that does not run is not sent anything, and its watchers run.
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
			failures[i] = d.refreshPlugin(ctx, id, slices.Contains(forced, id))
		})
	}
	group.Wait()
	return errors.Join(failures...)
}

func (d *Delivery) refreshPlugin(ctx context.Context, pluginID string, isForced bool) error {
	snapshot, isRunning, err := d.host.GetSnapshot(pluginID)
	if err != nil {
		return err
	}
	if !isRunning {
		if isForced {
			d.watchers.run(pluginID, d.host.GetPluginConfigRevision(pluginID))
		}
		return nil
	}
	if !d.ledger.isDue(pluginID) {
		return nil
	}
	return d.deliver(ctx, snapshot)
}

// Configure sends Configure to the plugin, or runs its watchers where it does not run. The caller holds Lock.
func (d *Delivery) Configure(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.host.GetSnapshot(pluginID)
	if err != nil {
		return err
	}
	if !isRunning {
		d.runWatchers(pluginID, d.ledger.isAwaitingFirstConfigure(pluginID))
		return nil
	}
	return d.deliver(ctx, snapshot)
}

func (d *Delivery) deliver(ctx context.Context, snapshot Snapshot) error {
	isFirst := d.ledger.isAwaitingFirstConfigure(snapshot.PluginID)
	isConfigured, sendErr := d.sendConfigure(ctx, snapshot)
	isCaller := sendErr != nil && isCallerFailure(ctx, sendErr)
	d.ledger.recordOutcome(snapshot.PluginID, sendErr, isCaller)
	if !isCaller {
		refused, _ := errors.AsType[*RefusedError](sendErr)
		d.host.RecordConfigureAnswer(snapshot.PluginID, refused)
	}
	if sendErr == nil && isConfigured {
		d.runWatchers(snapshot.PluginID, isFirst)
	}
	return sendErr
}

func (d *Delivery) runWatchers(pluginID string, isFirst bool) {
	revision := d.host.GetPluginConfigRevision(pluginID)
	if isFirst {
		d.watchers.runIfChanged(pluginID, revision)
		return
	}
	d.watchers.run(pluginID, revision)
}

func isCallerFailure(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	code := connect.CodeOf(err)
	return code == connect.CodeCanceled || code == connect.CodeDeadlineExceeded
}

func (d *Delivery) sendConfigure(ctx context.Context, snapshot Snapshot) (bool, error) {
	client := numenv1connect.NewPluginLifecycleServiceClient(
		d.host.GetClient(snapshot.Endpoint),
		transport.GetBaseURL(snapshot.Endpoint),
		connect.WithInterceptors(transport.AuthInterceptor(snapshot.Endpoint.Token)),
	)
	req := connect.NewRequest(&v1.ConfigureRequest{
		PluginId:          snapshot.PluginID,
		ConfigurationJson: snapshot.ConfigJSON,
		Values:            snapshot.Values,
	})
	callCtx, cancel := context.WithTimeout(ctx, ConfigureTimeout)
	defer cancel()
	_, err := client.Configure(callCtx, req)
	if err == nil {
		return true, nil
	}
	var connectErr *connect.Error
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented {
		return false, nil
	}
	return false, &RefusedError{PluginID: snapshot.PluginID, Text: getConfigureFailureText(err)}
}

func getConfigureFailureText(err error) string {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return "timed out"
	}
	if connectErr, isConnect := errors.AsType[*connect.Error](err); isConnect && connectErr.Message() != "" {
		return connectErr.Message()
	}
	return connect.CodeOf(err).String()
}
