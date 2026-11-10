package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ConfigPush gives the running plugins their configuration: the settings, the performance profile
// and the secrets they declare, each time one of them changes. It also keeps the secret
// declarations of the installed plugins, and drops the permissions of the ones that are gone.
type ConfigPush struct {
	registry *Registry
	sups     *supervision.Supervisors
	report   func(error)
	delivery *delivery.Delivery

	mu          sync.RWMutex
	performance string
	installed   []manifest.Discovered
	hasScanned  bool
}

func newConfigPush(registry *Registry, sups *supervision.Supervisors, report func(error)) *ConfigPush {
	c := &ConfigPush{registry: registry, sups: sups, report: report}
	c.delivery = delivery.New(delivery.Host{
		GetSnapshot:   c.getSnapshot,
		ListPluginIDs: registry.set.listIDs,
		ReportError:   report,
		GetClient:     transport.NewClientLookup(sups.Clients()).GetClient,
	})
	return c
}

// SetKeyStore sets where the values of secrets are kept.
func (c *ConfigPush) SetKeyStore(store pluginport.KeyStore) {
	c.delivery.SetKeyStore(store)
}

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (c *ConfigPush) SetSecretSource(source plugindomain.SecretSource) {
	c.delivery.SetSource(source)
	c.pruneSecretPermissions()
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (c *ConfigPush) ListDeclarations() []plugindomain.DeclarationKey {
	return c.listDeclarations(nil)
}

// Refresh sends Configure to each running plugin whose secrets differ from the last list it was
// given, and to each plugin in forced, and waits for every send. A secret operation calls it
// after its change is committed. The error names each plugin whose send failed.
// A plugin counts as given the list only after a send that read every value.
// If ctx ends or the call times out the plugin stays due; if the plugin answered with an error a plain refresh does not resend.
func (c *ConfigPush) Refresh(ctx context.Context, forced ...string) error {
	return c.delivery.Refresh(ctx, forced)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin while the delivery of secrets is locked.
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

func (c *ConfigPush) getSnapshot(pluginID string) (delivery.Snapshot, bool, error) {
	profile := c.getPerformanceProfile()
	endpoint, hasEndpoint := c.sups.GetEndpoint(pluginID)
	sup, hasSup := c.sups.Get(pluginID)
	if !hasEndpoint || !hasSup {
		return delivery.Snapshot{}, false, nil
	}
	if state := sup.GetState(); state != lifecycle.StateActive && state != lifecycle.StateIdle {
		return delivery.Snapshot{}, false, nil
	}
	rawSettings := c.registry.set.getPluginSettings(pluginID)
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
		Declared:   c.listActiveDeclarations(pluginID, rawSettings),
		Requested:  plugindomain.ListAskedSecretGroups(c.listDeclarations(nil)),
	}, true, nil
}

func (c *ConfigPush) listActiveDeclarations(pluginID string, values map[string]any) []manifest.SecretDeclaration {
	p, ok := c.registry.set.getKnown(pluginID)
	if !ok {
		return nil
	}
	var active []manifest.SecretDeclaration
	for _, declaration := range p.Manifest.Secrets {
		if p.Manifest.IsSecretActive(declaration, values) {
			active = append(active, declaration)
		}
	}
	return active
}

// listDeclarations lists the secret declarations of the installed plugins, then of the plugins the host
// knows that are not among them.
func (c *ConfigPush) listDeclarations(installed []manifest.Discovered) []plugindomain.DeclarationKey {
	v, release := c.registry.set.view()
	defer release()
	var declarations []plugindomain.DeclarationKey
	installedIDs := make(map[string]bool, len(installed))
	for _, p := range installed {
		installedIDs[p.Manifest.ID] = true
		declarations = appendDeclarations(declarations, p)
	}
	for _, p := range v.ordered() {
		if !installedIDs[p.Manifest.ID] {
			declarations = appendDeclarations(declarations, p)
		}
	}
	return declarations
}

func appendDeclarations(declarations []plugindomain.DeclarationKey, p manifest.Discovered) []plugindomain.DeclarationKey {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, plugindomain.DeclarationKey{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}

// recordScan keeps the plugins a scan of the folders found, whether or not they target this host.
func (c *ConfigPush) recordScan(installed []manifest.Discovered) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.installed = installed
	c.hasScanned = true
}

func (c *ConfigPush) pruneSecretPermissions() {
	prune := c.delivery.GetSource().Prune
	c.mu.RLock()
	skip := !c.hasScanned || c.registry.set.hasProblems()
	installed := c.installed
	c.mu.RUnlock()
	if prune == nil || skip {
		return
	}
	if err := prune(c.listDeclarations(installed)); err != nil {
		c.report(fmt.Errorf("drop the permissions of removed declarations: %w", err))
	}
}
