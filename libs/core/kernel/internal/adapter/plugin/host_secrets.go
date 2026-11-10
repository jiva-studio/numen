package plugin

import (
	"context"
	"fmt"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	pluginport "github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// SetKeyStore sets where the values of secrets are kept.
func (h *Host) SetKeyStore(store pluginport.KeyStore) {
	h.secrets.setKeyStore(store)
}

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (h *Host) SetSecretSource(source plugindomain.SecretSource) {
	h.secrets.setSource(source)
	h.pruneSecretPermissions()
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (h *Host) ListDeclarations() []plugindomain.DeclarationKey {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.listDeclarationsLocked()
}

func (h *Host) listDeclarationsLocked() []plugindomain.DeclarationKey {
	var declarations []plugindomain.DeclarationKey
	for _, p := range h.getPluginsInOrderLocked() {
		declarations = appendDeclarations(declarations, p)
	}
	return declarations
}

// Refresh sends Configure to each running plugin whose secrets differ from the last list it was
// given, and to each plugin in forced, and waits for every send. A secret operation calls it
// after its change is committed. The error names each plugin whose send failed.
// A plugin counts as given the list only after a send that read every value.
// If ctx ends or the call times out the plugin stays due; if the plugin answered with an error a plain refresh does not resend.
func (h *Host) Refresh(ctx context.Context, forced ...string) error {
	return h.secrets.refresh(ctx, forced)
}

// WatchPluginConfig registers fn to run each time the plugin has been given its configuration.
// fn runs on its own goroutine, so it may call the plugin while the delivery of secrets is locked.
// The returned function removes the registration.
func (h *Host) WatchPluginConfig(pluginID string, fn func()) (stop func()) {
	return h.secrets.watchPluginConfig(pluginID, fn)
}

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

func (h *Host) listInstalledDeclarationsLocked() []plugindomain.DeclarationKey {
	var declarations []plugindomain.DeclarationKey
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

func appendDeclarations(declarations []plugindomain.DeclarationKey, p manifest.Discovered) []plugindomain.DeclarationKey {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, plugindomain.DeclarationKey{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}
