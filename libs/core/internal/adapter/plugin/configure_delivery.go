package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
)

const configureCallTimeout = 3 * time.Second

// configSnapshot is what a Configure call to one plugin is made of, read while mu is held.
type configSnapshot struct {
	pluginID   string
	endpoint   transport.Endpoint
	configJSON string
	values     map[string]string
	getExtra   func() configureExtra
}

// configureExtra is the part of a Configure call that its owner adds to the settings.
type configureExtra struct {
	signature           string
	isComplete          bool
	isConfigureOptional bool
	read                func(ctx context.Context) ([]*v1.ConfigureSecrets, error)
}

func (s configSnapshot) resolveExtra() configureExtra {
	if s.getExtra == nil {
		return configureExtra{isComplete: true, isConfigureOptional: true}
	}
	return s.getExtra()
}

type configureHost struct {
	getConfigSnapshot func(pluginID string) (configSnapshot, bool, error)
	listPluginIDs     func() []string
}

// configureDelivery sends Configure to the plugins: their settings, the performance profile and
// what the owner of an extra adds. It remembers what each plugin was given last.
type configureDelivery struct {
	host                   configureHost
	lock                   chan struct{}
	mu                     sync.RWMutex
	delivered              map[string]string
	rejected               map[string]string
	awaitingFirstConfigure map[string]bool
	configWatchers         map[string]map[int]func()
	nextWatcher            int
}

func newConfigureDelivery(host configureHost) *configureDelivery {
	return &configureDelivery{
		host:                   host,
		lock:                   make(chan struct{}, 1),
		delivered:              make(map[string]string),
		rejected:               make(map[string]string),
		awaitingFirstConfigure: make(map[string]bool),
	}
}

func (d *configureDelivery) lockDelivery(ctx context.Context) error {
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

func (d *configureDelivery) unlockDelivery() { <-d.lock }

func (d *configureDelivery) markAwaitingFirstConfigure(pluginID string) {
	d.mu.Lock()
	delete(d.delivered, pluginID)
	delete(d.rejected, pluginID)
	d.awaitingFirstConfigure[pluginID] = true
	d.mu.Unlock()
}

func (d *configureDelivery) clearAwaitingFirstConfigure(pluginID string) {
	d.mu.Lock()
	delete(d.awaitingFirstConfigure, pluginID)
	d.mu.Unlock()
}

func (d *configureDelivery) isAwaitingFirstConfigure(pluginID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.awaitingFirstConfigure[pluginID]
}

func (d *configureDelivery) forget(pluginIDs []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, id := range pluginIDs {
		delete(d.delivered, id)
		delete(d.rejected, id)
	}
}

func (d *configureDelivery) refresh(ctx context.Context, forced []string) error {
	d.forget(forced)
	if err := d.lockDelivery(ctx); err != nil {
		return err
	}
	defer d.unlockDelivery()
	ids := d.host.listPluginIDs()
	failures := make([]error, len(ids))
	var group sync.WaitGroup
	for i, id := range ids {
		if d.isAwaitingFirstConfigure(id) {
			continue
		}
		group.Go(func() {
			failures[i] = d.refreshPlugin(ctx, id)
		})
	}
	group.Wait()
	return errors.Join(failures...)
}

func (d *configureDelivery) refreshPlugin(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.host.getConfigSnapshot(pluginID)
	if err != nil {
		return err
	}
	if !isRunning {
		return nil
	}
	extra := snapshot.resolveExtra()
	if extra.isComplete && !d.isDue(pluginID, extra.signature) {
		return nil
	}
	return d.deliver(ctx, snapshot, extra)
}

func (d *configureDelivery) configurePlugin(ctx context.Context, pluginID string) error {
	snapshot, isRunning, err := d.host.getConfigSnapshot(pluginID)
	if !isRunning || err != nil {
		return err
	}
	return d.deliver(ctx, snapshot, snapshot.resolveExtra())
}

func (d *configureDelivery) isDue(pluginID, signature string) bool {
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

func (d *configureDelivery) deliver(ctx context.Context, snapshot configSnapshot, extra configureExtra) error {
	var entries []*v1.ConfigureSecrets
	var readErr error
	if extra.read != nil {
		entries, readErr = extra.read(ctx)
	}
	isConfigured, sendErr := d.sendConfigure(ctx, snapshot, entries, extra.isConfigureOptional)
	d.recordOutcome(ctx, snapshot.pluginID, extra.signature, sendErr, readErr)
	if sendErr == nil && isConfigured {
		d.runConfigWatchers(snapshot.pluginID)
	}
	return errors.Join(sendErr, readErr)
}

func (d *configureDelivery) recordOutcome(ctx context.Context, pluginID, signature string, sendErr, readErr error) {
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

func (d *configureDelivery) sendConfigure(
	ctx context.Context, snapshot configSnapshot, entries []*v1.ConfigureSecrets, isConfigureOptional bool,
) (bool, error) {
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
	if errors.As(err, &connectErr) && connectErr.Code() == connect.CodeUnimplemented && isConfigureOptional {
		return false, nil
	}
	return false, fmt.Errorf("configure plugin %s: %s", snapshot.pluginID, getConfigureFailureText(err))
}

func (d *configureDelivery) watchPluginConfig(pluginID string, fn func()) (stop func()) {
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

func (d *configureDelivery) runConfigWatchers(pluginID string) {
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
