package plugin

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetCompletionChoice returns the agent and the model the plugin's setting of
// type agent holds, as the person stored it. A plugin declaring no such setting,
// or one never set, holds the empty choice.
func (h *Host) GetCompletionChoice(pluginID string) port.CompletionChoice {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[pluginID]
	if !ok {
		return port.CompletionChoice{}
	}
	key, ok := p.Manifest.GetAgentSettingKey()
	if !ok {
		return port.CompletionChoice{}
	}
	return port.ReadCompletionChoice(h.settings[pluginID][key])
}

// CheckCompletionConsent answers nil for a plugin the person lets ask for model
// completions, and port.ErrNoConsent for any other. A plugin the person has not
// answered for is recorded as asking, so the window can put the question to them.
func (h *Host) CheckCompletionConsent(pluginID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	allowed, isAnswered := h.settings[pluginID][port.SettingAIConsent].(bool)
	switch {
	case isAnswered && allowed:
		return nil
	case !isAnswered:
		h.consentAsks[pluginID] = true
	}
	return port.ErrNoConsent
}

// getCompletionConsentLocked is whether the person lets the plugin ask for model
// completions, for a caller holding h.mu.
func (h *Host) getCompletionConsentLocked(manifest port.PluginManifest) port.CompletionConsent {
	if !manifest.HasPermission(port.PermissionAIComplete) {
		return port.ConsentNone
	}
	allowed, isAnswered := h.settings[manifest.ID][port.SettingAIConsent].(bool)
	switch {
	case isAnswered && allowed:
		return port.ConsentAllowed
	case isAnswered:
		return port.ConsentRefused
	case h.consentAsks[manifest.ID]:
		return port.ConsentAsked
	}
	return port.ConsentUnanswered
}
