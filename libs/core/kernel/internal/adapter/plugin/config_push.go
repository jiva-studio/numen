package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ConfigPush gives the running plugins their configuration: the settings, the performance profile
// and the secrets they declare, each time one of them changes.
type ConfigPush struct {
	t        *table
	errs     *errorSink
	settings *Settings
	delivery *secretDelivery
	prune    func()
}

func newConfigPush(t *table, errs *errorSink, settings *Settings) *ConfigPush {
	c := &ConfigPush{t: t, errs: errs, settings: settings}
	c.delivery = newSecretDelivery(secretHost{
		getConfigSnapshot: c.getConfigSnapshot,
		listPluginIDs:     c.listPluginIDs,
		reportError:       errs.report,
		getClient:         t.clients.Get,
	})
	return c
}

// SetKeyStore sets where the values of secrets are kept.
func (c *ConfigPush) SetKeyStore(store pluginport.KeyStore) {
	c.delivery.setKeyStore(store)
}

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (c *ConfigPush) SetSecretSource(source plugindomain.SecretSource) {
	c.delivery.setSource(source)
	c.prune()
}

func (c *ConfigPush) getSecretSource() plugindomain.SecretSource {
	return c.delivery.getSource()
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (c *ConfigPush) ListDeclarations() []plugindomain.DeclarationKey {
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	return c.listDeclarationsLocked()
}

func (c *ConfigPush) listDeclarationsLocked() []plugindomain.DeclarationKey {
	var declarations []plugindomain.DeclarationKey
	for _, p := range c.t.getPluginsInOrderLocked() {
		declarations = appendDeclarations(declarations, p)
	}
	return declarations
}

// Refresh sends Configure to each running plugin whose secrets differ from the last list it was
// given, and to each plugin in forced, and waits for every send. A secret operation calls it
// after its change is committed. The error names each plugin whose send failed.
// A plugin counts as given the list only after a send that read every value.
// If ctx ends or the call times out the plugin stays due; if the plugin answered with an error a plain refresh does not resend.
func (c *ConfigPush) Refresh(ctx context.Context, forced ...string) error {
	return c.delivery.refresh(ctx, forced)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin while the delivery of secrets is locked.
// The returned function removes the registration.
func (c *ConfigPush) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return c.delivery.watchPluginConfig(pluginID, fn)
}

// NotifyPluginConfig sends the resolved configuration to a running plugin process.
func (c *ConfigPush) NotifyPluginConfig(ctx context.Context, pluginID string) error {
	if err := c.delivery.lockSecrets(ctx); err != nil {
		return err
	}
	defer c.delivery.unlockSecrets()
	return c.delivery.configurePlugin(ctx, pluginID)
}

func (c *ConfigPush) awaitFirstConfigure(pluginID string) {
	c.delivery.markAwaitingFirstConfigure(pluginID)
}

func (c *ConfigPush) configureActivated(ctx context.Context, pluginID string) error {
	if lockErr := c.delivery.lockSecrets(ctx); lockErr != nil {
		c.delivery.clearAwaitingFirstConfigure(pluginID)
		return lockErr
	}
	configureErr := c.delivery.configurePlugin(ctx, pluginID)
	c.delivery.clearAwaitingFirstConfigure(pluginID)
	c.delivery.unlockSecrets()
	if configureErr != nil {
		c.errs.report(fmt.Errorf("push settings to plugin %s: %w", pluginID, configureErr))
	}
	return nil
}

func (c *ConfigPush) listPluginIDs() []string {
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	ids := make([]string, 0, len(c.t.plugins))
	for _, p := range c.t.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	return ids
}

// configSnapshot is what a Configure call to one plugin is made of, read while mu is held.
type configSnapshot struct {
	pluginID   string
	endpoint   plugindomain.Endpoint
	configJSON string
	values     map[string]string
	declared   []manifest.SecretDeclaration
	requested  []string
	store      pluginport.KeyStore
	source     plugindomain.SecretSource
}

func (c *ConfigPush) getConfigSnapshot(pluginID string) (configSnapshot, bool, error) {
	profile := c.settings.getPerformanceProfile()
	c.t.mu.RLock()
	defer c.t.mu.RUnlock()
	endpoint, hasEndpoint := c.t.endpoints[pluginID]
	sup, hasSup := c.t.supervisors[pluginID]
	if !hasEndpoint || !hasSup {
		return configSnapshot{}, false, nil
	}
	if state := sup.GetState(); state != lifecycle.StateActive && state != lifecycle.StateIdle {
		return configSnapshot{}, false, nil
	}
	rawSettings := c.t.getPluginSettingsLocked(pluginID)
	cfgJSON, err := json.Marshal(rawSettings)
	if err != nil {
		return configSnapshot{}, true, fmt.Errorf("marshal plugin %s settings: %w", pluginID, err)
	}
	values := plugindomain.FormatSettingValues(rawSettings)
	if profile != "" {
		values[plugindomain.ConfigPerformanceProfile] = profile
	}
	return configSnapshot{
		pluginID:   pluginID,
		endpoint:   endpoint,
		configJSON: string(cfgJSON),
		values:     values,
		declared:   c.listActiveDeclarationsLocked(pluginID, rawSettings),
		requested:  plugindomain.ListAskedSecretGroups(c.listDeclarationsLocked()),
	}, true, nil
}

func (c *ConfigPush) listActiveDeclarationsLocked(pluginID string, values map[string]any) []manifest.SecretDeclaration {
	p, ok := c.t.plugins[pluginID]
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

func appendDeclarations(declarations []plugindomain.DeclarationKey, p manifest.Discovered) []plugindomain.DeclarationKey {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, plugindomain.DeclarationKey{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}
