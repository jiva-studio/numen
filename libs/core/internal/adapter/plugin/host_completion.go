package plugin

import (
	completionport "github.com/jiva-studio/numen/libs/core/usecase/completion/completioncontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

// GetCompletionChoice returns the agent and the model the plugin's setting of
// type agent holds, as the person stored it. A plugin declaring no such setting,
// or one never set, holds the empty choice.
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

// CheckCompletionConsent answers nil for a plugin the person lets ask for model
// completions, and port.ErrNoConsent for any other. A plugin the person has not
// answered for is recorded as asking, so the window can put the question to them.
func (h *Host) CheckCompletionConsent(pluginID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	allowed, isAnswered := h.settings[pluginID][pluginport.SettingAIConsent].(bool)
	switch {
	case isAnswered && allowed:
		return nil
	case !isAnswered:
		h.consentAsks[pluginID] = true
	}
	return completionport.ErrNoConsent
}

// getCompletionConsentLocked is whether the person lets the plugin ask for model
// completions, for a caller holding h.mu.
func (h *Host) getCompletionConsentLocked(manifest pluginport.PluginManifest) pluginport.CompletionConsent {
	if !manifest.HasPermission(pluginport.PermissionAIComplete) {
		return pluginport.ConsentNone
	}
	allowed, isAnswered := h.settings[manifest.ID][pluginport.SettingAIConsent].(bool)
	switch {
	case isAnswered && allowed:
		return pluginport.ConsentAllowed
	case isAnswered:
		return pluginport.ConsentRefused
	case h.consentAsks[manifest.ID]:
		return pluginport.ConsentAsked
	}
	return pluginport.ConsentUnanswered
}
