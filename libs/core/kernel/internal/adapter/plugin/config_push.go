package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// configPlugins is what the push reads of the set of plugins.
type configPlugins interface {
	listIDs() []string
	getPluginSettings(pluginID string) map[string]any
	getKnownPlugin(pluginID string) (manifest.Discovered, bool)
	view() (setView, func())
	hasProblems() bool
}

// ConfigPush gives the running plugins their configuration: the settings and the performance profile,
// each time one of them changes.
type ConfigPush struct {
	set      configPlugins
	sups     *supervision.Supervisors
	report   func(error)
	delivery *delivery.Delivery

	mu          sync.RWMutex
	performance string
}

// ConfigPushDeps are what a ConfigPush reads and where it reports an error.
type ConfigPushDeps struct {
	Set         *PluginSet
	Supervisors *supervision.Supervisors
	Report      func(error)
}

func NewConfigPush(d ConfigPushDeps) *ConfigPush {
	c := &ConfigPush{set: d.Set, sups: d.Supervisors, report: d.Report}
	c.delivery = delivery.New(c)
	return c
}

// ListPluginIDs returns the ids of the plugins the host knows.
func (c *ConfigPush) ListPluginIDs() []string {
	return c.set.listIDs()
}

// GetClient returns the client that calls the plugin process an endpoint belongs to.
func (c *ConfigPush) GetClient(endpoint transport.Endpoint) *http.Client {
	return transport.NewClientLookup(c.sups.Clients()).GetClient(endpoint)
}

// Refresh sends Configure to each running plugin that was not given its configuration yet, and to
// each plugin in forced, and waits for every send. An edit of a connection calls it after the change
// is committed. The error names each plugin whose send failed.
// If ctx ends or the call times out the plugin stays due; if the plugin answered with an error a plain refresh does not resend.
func (c *ConfigPush) Refresh(ctx context.Context, forced ...string) error {
	return c.delivery.Refresh(ctx, forced)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin while the delivery of Configure is locked.
// The returned function removes the registration.
func (c *ConfigPush) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return c.delivery.WatchConfig(pluginID, fn)
}

// NotifyPluginConfig sends the resolved configuration to a running plugin process.
func (c *ConfigPush) NotifyPluginConfig(ctx context.Context, pluginID string) error {
	if err := c.delivery.Lock(ctx); err != nil {
		return err
	}
	defer c.delivery.Unlock()
	return c.delivery.Configure(ctx, pluginID)
}

// SetPerformanceProfile records the profile of the performance setting and pushes it to every
// running plugin when it changed. A plugin that is not running receives it with its first
// Configure call.
func (c *ConfigPush) SetPerformanceProfile(ctx context.Context, profile string) {
	c.mu.Lock()
	if c.performance == profile {
		c.mu.Unlock()
		return
	}
	c.performance = profile
	c.mu.Unlock()

	for _, id := range c.sups.ListEndpointOwners() {
		if err := c.NotifyPluginConfig(ctx, id); err != nil {
			c.report(fmt.Errorf("push the performance profile to plugin %s: %w", id, err))
		}
	}
}

func (c *ConfigPush) getPerformanceProfile() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.performance
}

func (c *ConfigPush) awaitFirstConfigure(pluginID string) {
	c.delivery.AwaitFirstConfigure(pluginID)
}

func (c *ConfigPush) configureActivated(ctx context.Context, pluginID string) error {
	if lockErr := c.delivery.Lock(ctx); lockErr != nil {
		c.delivery.ClearFirstConfigure(pluginID)
		return lockErr
	}
	configureErr := c.delivery.Configure(ctx, pluginID)
	c.delivery.ClearFirstConfigure(pluginID)
	c.delivery.Unlock()
	if configureErr != nil {
		c.report(fmt.Errorf("push settings to plugin %s: %w", pluginID, configureErr))
	}
	return nil
}

func (c *ConfigPush) GetSnapshot(pluginID string) (delivery.Snapshot, bool, error) {
	profile := c.getPerformanceProfile()
	endpoint, hasEndpoint := c.sups.GetEndpoint(pluginID)
	sup, hasSup := c.sups.Get(pluginID)
	if !hasEndpoint || !hasSup {
		return delivery.Snapshot{}, false, nil
	}
	if state := sup.GetState(); state != lifecycle.StateActive && state != lifecycle.StateIdle {
		return delivery.Snapshot{}, false, nil
	}
	rawSettings := c.set.getPluginSettings(pluginID)
	cfgJSON, err := json.Marshal(rawSettings)
	if err != nil {
		return delivery.Snapshot{}, true, fmt.Errorf("marshal plugin %s settings: %w", pluginID, err)
	}
	values := plugindomain.FormatSettingValues(rawSettings)
	if profile != "" {
		values[plugindomain.ConfigPerformanceProfile] = profile
	}
	return delivery.Snapshot{
		PluginID:   pluginID,
		Endpoint:   endpoint,
		ConfigJSON: string(cfgJSON),
		Values:     values,
	}, true, nil
}
