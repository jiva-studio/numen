package plugin

import (
	"fmt"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// SetSecretSource sets where the list of secrets is read from and how the permissions of removed
// declarations are dropped.
func (h *Host) SetSecretSource(source pluginport.SecretSource) {
	h.secrets.setSource(source)
	h.pruneSecretPermissions()
}

// ListDeclarations returns every secret declaration of every installed plugin, in the order of the
// plugin ids and then of the manifest.
func (h *Host) ListDeclarations() []pluginport.DeclarationKey {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.listDeclarationsLocked()
}

func (h *Host) listDeclarationsLocked() []pluginport.DeclarationKey {
	var declarations []pluginport.DeclarationKey
	for _, p := range h.getPluginsInOrderLocked() {
		declarations = appendDeclarations(declarations, p)
	}
	return declarations
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

func (h *Host) listInstalledDeclarationsLocked() []pluginport.DeclarationKey {
	var declarations []pluginport.DeclarationKey
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

func appendDeclarations(declarations []pluginport.DeclarationKey, p manifest.Discovered) []pluginport.DeclarationKey {
	for _, declaration := range p.Manifest.Secrets {
		declarations = append(declarations, pluginport.DeclarationKey{
			Plugin: p.Manifest.ID, Name: declaration.Name, Group: declaration.Group,
		})
	}
	return declarations
}
