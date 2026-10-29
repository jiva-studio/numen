package plugin

import (
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// GetCompletionChoice returns the agent and the model stored by the user in the
// plugin's setting of type agent. A plugin declaring no such setting, or one
// never set, yields the empty choice.
func (h *Host) GetCompletionChoice(pluginID string) completionport.CompletionChoice {
	h.mu.RLock()
	defer h.mu.RUnlock()
	p, ok := h.plugins[pluginID]
	if !ok {
		return completionport.CompletionChoice{}
	}
	key, ok := p.Manifest.GetAgentSettingKey()
	if !ok {
		return completionport.CompletionChoice{}
	}
	return completionport.ReadCompletionChoice(h.settings[pluginID][key])
}

// CheckCompletionConsent returns nil for a plugin the user allows to request
// model completions, and port.ErrNoConsent for any other. A plugin with no
// recorded answer is recorded as asking, so the window can put the question to the user.
func (h *Host) CheckCompletionConsent(pluginID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	allowed, isSet := h.settings[pluginID][pluginport.SettingAIConsent].(bool)
	switch {
	case isSet && allowed:
		return nil
	case !isSet:
		h.consentAsks[pluginID] = true
	}
	return completionport.ErrNoConsent
}

// getCompletionConsentLocked reports whether the user allows the plugin to
// request model completions, for a caller holding h.mu.
func (h *Host) getCompletionConsentLocked(manifest pluginport.PluginManifest) pluginport.CompletionConsent {
	if !manifest.HasPermission(pluginport.PermissionAIComplete) {
		return pluginport.ConsentNone
	}
	allowed, isSet := h.settings[manifest.ID][pluginport.SettingAIConsent].(bool)
	switch {
	case isSet && allowed:
		return pluginport.ConsentAllowed
	case isSet:
		return pluginport.ConsentDenied
	case h.consentAsks[manifest.ID]:
		return pluginport.ConsentRequested
	}
	return pluginport.ConsentPending
}
