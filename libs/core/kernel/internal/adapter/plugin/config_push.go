package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/machine"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// configPlugins is what the push reads of the set of plugins.
type configPlugins interface {
	getKnownPlugin(pluginID string) (manifest.Discovered, bool)
	listIDs() []string
	getPluginSettings(pluginID string) map[string]any
	isPluginEnabled(pluginID string) bool
}

// ConfigPush gives the running plugins their configuration: the settings and the performance profile,
// each time one of them changes.
type ConfigPush struct {
	set      configPlugins
	sups     *supervision.Supervisors
	report   func(error)
	health   supervision.HealthItems
	delivery *delivery.Delivery

	mu          sync.RWMutex
	performance plugindomain.Performance
}

// ConfigPushDeps are what a ConfigPush reads, where it reports an error and the list it keeps the
// refusals of the plugins in, as the item plugin.HealthKeyConfigure of each plugin.
type ConfigPushDeps struct {
	Set         *PluginSet
	Supervisors *supervision.Supervisors
	Report      func(error)
	Health      supervision.HealthItems
}

func NewConfigPush(d ConfigPushDeps) *ConfigPush {
	c := &ConfigPush{set: d.Set, sups: d.Supervisors, report: d.Report, health: d.Health}
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

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration, and each
// time its configuration changes while it does not run and is enabled. A started plugin given the
// configuration fn last ran for does not run fn.
// fn runs on its own goroutine, so it may call the plugin while the delivery of Configure is locked.
// The returned function removes the registration.
func (c *ConfigPush) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return c.delivery.WatchConfig(pluginID, func() {
		if c.set.isPluginEnabled(pluginID) {
			fn()
		}
	})
}

// GetPluginConfigRevision returns a digest of what the plugin is configured with: its version, its settings
// without the enabled flag, and the performance limits. Equal revisions are one configuration, in this
// process or another.
func (c *ConfigPush) GetPluginConfigRevision(pluginID string) string {
	settings := maps.Clone(c.set.getPluginSettings(pluginID))
	delete(settings, settingEnabled)
	encoded, err := json.Marshal(settings)
	if err != nil {
		return ""
	}
	discovered, _ := c.set.getKnownPlugin(pluginID)
	performance := c.getPerformance(discovered.Manifest.Compute)
	parts := []string{
		discovered.Manifest.Version, string(encoded),
		performance.Profile, performance.Provider, strconv.Itoa(performance.Threads), strconv.Itoa(performance.Nice),
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(digest[:])
}

// NotifyPluginConfig sends the resolved configuration to a running plugin process.
func (c *ConfigPush) NotifyPluginConfig(ctx context.Context, pluginID string) error {
	if err := c.delivery.Lock(ctx); err != nil {
		return err
	}
	defer c.delivery.Unlock()
	return c.configurePlugin(ctx, pluginID)
}

func (c *ConfigPush) configurePlugin(ctx context.Context, pluginID string) error {
	if sup, ok := c.sups.Get(pluginID); ok {
		if err := sup.Child.SetNice(c.GetProcessNice()); err != nil {
			c.report(fmt.Errorf("set priority of %s: %w", pluginID, err))
		}
	}
	return c.delivery.Configure(ctx, pluginID)
}

// SetPerformance records the limits of the performance profile and pushes them to every
// running plugin when they changed. A plugin that is not running receives them with its first
// Configure call.
func (c *ConfigPush) SetPerformance(ctx context.Context, performance plugindomain.Performance) {
	c.mu.Lock()
	if c.performance == performance {
		c.mu.Unlock()
		return
	}
	c.performance = performance
	c.mu.Unlock()

	for _, id := range c.sups.ListEndpointOwners() {
		if err := c.NotifyPluginConfig(ctx, id); err != nil {
			c.report(fmt.Errorf("push the performance limits to plugin %s: %w", id, err))
		}
	}
}

// GetProcessNice returns the configured niceness for a newly launched plugin.
func (c *ConfigPush) GetProcessNice() int {
	profile := c.getPerformance(nil)
	if profile.Profile == "" {
		return machine.ResolveCompute(machine.ProfileBalanced, machine.Specs{}).Nice
	}
	return profile.Nice
}

func (c *ConfigPush) getPerformance(compute *manifest.Compute) plugindomain.Performance {
	c.mu.RLock()
	defer c.mu.RUnlock()
	performance := c.performance
	if compute != nil && !compute.Threads {
		performance.Threads = 0
	}
	if compute == nil || !compute.Accelerator {
		performance.Provider = ""
	}
	return performance
}

func (c *ConfigPush) awaitFirstConfigure(pluginID string) {
	c.delivery.AwaitFirstConfigure(pluginID)
}

func (c *ConfigPush) configureActivated(ctx context.Context, pluginID string) error {
	if lockErr := c.delivery.Lock(ctx); lockErr != nil {
		c.delivery.ClearFirstConfigure(pluginID)
		return lockErr
	}
	configureErr := c.configurePlugin(ctx, pluginID)
	c.delivery.ClearFirstConfigure(pluginID)
	c.delivery.Unlock()
	// A refusal of the plugin is listed as its configure item.
	if refused, _ := errors.AsType[*delivery.RefusedError](configureErr); configureErr != nil && refused == nil {
		c.report(fmt.Errorf("push settings to plugin %s: %w", pluginID, configureErr))
	}
	return nil
}

// RecordConfigureAnswer lists the refusal of a Configure call as the configure item of the plugin, with
// the message of the plugin as its text, and takes the item out once a call is answered without one.
func (c *ConfigPush) RecordConfigureAnswer(pluginID string, refused *delivery.RefusedError) {
	if refused == nil {
		c.health.Remove(pluginID, plugindomain.HealthKeyConfigure)
		return
	}
	text, detail := plugindomain.FitHealthText(refused.Text)
	c.health.Set(plugindomain.HealthItem{
		PluginID: pluginID,
		Key:      plugindomain.HealthKeyConfigure,
		Severity: plugindomain.HealthError,
		Text:     text,
		Detail:   detail,
		Action: &plugindomain.HealthAction{
			Command: plugindomain.AppCommandOpenSettings,
			Values:  map[string]string{plugindomain.OpenSettingsPlugin: pluginID},
		},
	})
}

func (c *ConfigPush) GetSnapshot(pluginID string) (delivery.Snapshot, bool, error) {
	discovered, _ := c.set.getKnownPlugin(pluginID)
	performance := c.getPerformance(discovered.Manifest.Compute)
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
	maps.Copy(values, performance.GetConfigValues())
	return delivery.Snapshot{
		PluginID:   pluginID,
		Endpoint:   endpoint,
		ConfigJSON: string(cfgJSON),
		Values:     values,
	}, true, nil
}
