package plugin

import (
	"context"
	"fmt"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// SetKeyStore sets where the values of secrets are kept.
func (h *Host) SetKeyStore(store pluginport.KeyStore) {
	h.secrets.setKeyStore(store)
}

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (h *Host) SetSecretSource(source pluginport.SecretSource) {
	h.secrets.setSource(source)
	h.pruneSecretPermissions()
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (h *Host) ListDeclarations() []pluginport.SecretPermission {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.listDeclarationsLocked()
}

func (h *Host) listDeclarationsLocked() []pluginport.SecretPermission {
	var declarations []pluginport.SecretPermission
	for _, p := range h.getPluginsInOrderLocked() {
		declarations = appendDeclarations(declarations, p)
	}
	return declarations
}

// ApplySecretChange runs change while no other secret operation runs. It then sends Configure to
// each running plugin whose list of secrets differs from the one it had before change, and to each
// plugin whose id change returns, and waits for every send. When a send fails the change stays
// committed and the error names the plugin. It returns the context error, with change not run, when
// ctx ends before the lock is free.
func (h *Host) ApplySecretChange(ctx context.Context, change func() (readers []string, err error)) error {
	return h.secrets.applyChange(ctx, change)
}

// RefreshSecrets sends Configure to each running plugin whose list of secrets differs from the one
// it was last given.
func (h *Host) RefreshSecrets(ctx context.Context) error {
	return h.secrets.refresh(ctx)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin, which a caller holding the lock of the
// secrets could not do. The returned function removes the registration.
func (h *Host) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return h.secrets.watchPluginConfig(pluginID, fn)
}

// listPluginIDs returns the ids of the registered plugins in order.
func (h *Host) listPluginIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.plugins))
	for _, p := range h.getPluginsInOrderLocked() {
		ids = append(ids, p.Manifest.ID)
	}
	return ids
}

func (h *Host) pruneSecretPermissions() {
	prune := h.secrets.getSource().Prune
	h.mu.RLock()
	skip := !h.hasScanned || len(h.problems) > 0
	declared := h.listInstalledDeclarationsLocked()
	h.mu.RUnlock()
	if prune == nil || skip {
		return
	}
	if err := prune(declared); err != nil {
		h.reportError(fmt.Errorf("drop the permissions of removed declarations: %w", err))
	}
}

// listInstalledDeclarationsLocked returns the declarations of every plugin installed in the plugin
// folders whatever its target, and of the registered plugins that are not on disk. A host of one
// target must not drop the permissions of a plugin that targets another window.
func (h *Host) listInstalledDeclarationsLocked() []pluginport.SecretPermission {
	var declarations []pluginport.SecretPermission
	installed := make(map[string]bool, len(h.installed))
	for _, p := range h.installed {
		installed[p.Manifest.ID] = true
		declarations = appendDeclarations(declarations, p)
	}
	for _, p := range h.getPluginsInOrderLocked() {
		if !installed[p.Manifest.ID] {
			declarations = appendDeclarations(declarations, p)
		}
	}
	return declarations
}

func appendDeclarations(declarations []pluginport.SecretPermission, p manifest.Discovered) []pluginport.SecretPermission {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, pluginport.SecretPermission{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}
