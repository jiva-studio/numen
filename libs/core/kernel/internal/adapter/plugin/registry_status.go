package plugin

import (
	"slices"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// PluginStatus describes the enablement and runtime status of a plugin.
type PluginStatus struct {
	ID            string `json:"id"`
	Dir           string `json:"dir,omitempty"`
	Enabled       bool   `json:"enabled"`
	IsActive      bool   `json:"isActive"`
	Failed        bool   `json:"failed"`
	FailureReason string `json:"failureReason,omitempty"`
}

func (r *Registry) GetPluginStatus(pluginID string) (PluginStatus, bool) {
	v, release := r.set.view()
	defer release()
	if _, ok := v.get(pluginID); !ok {
		return PluginStatus{}, false
	}
	return r.getStatus(v, pluginID), true
}

func (r *Registry) getStatus(v setView, pluginID string) PluginStatus {
	status := PluginStatus{
		ID:            pluginID,
		Enabled:       v.isEnabled(pluginID),
		IsActive:      r.sups.IsActive(pluginID),
		Failed:        false,
		FailureReason: "",
		Dir:           "",
	}
	if sup, ok := r.sups.Get(pluginID); ok {
		if cause := sup.Failure.GetCause(); cause != nil {
			status.Failed = true
			status.FailureReason = cause.Error()
		}
	}
	return status
}

// GetPluginStatuses returns statuses for all registered plugins, in the order of their ids.
func (r *Registry) GetPluginStatuses() []PluginStatus {
	v, release := r.set.view()
	defer release()
	ordered := v.ordered()
	statuses := make([]PluginStatus, 0, len(ordered)+len(v.listProblems()))
	for _, p := range ordered {
		statuses = append(statuses, r.getStatus(v, p.Manifest.ID))
	}
	for _, problem := range v.listProblems() {
		statuses = append(statuses, PluginStatus{
			ID:            "",
			Dir:           problem.Dir,
			Enabled:       false,
			IsActive:      false,
			Failed:        true,
			FailureReason: problem.Err.Error(),
		})
	}
	return statuses
}

// GetPluginInfos returns specifications of all registered plugins and their settings schemas, in
// the order of their ids.
func (r *Registry) GetPluginInfos() []plugindomain.Info {
	v, release := r.set.view()
	defer release()
	enabled := v.enabled()
	ordered := v.ordered()
	result := make([]plugindomain.Info, 0, len(ordered))
	for _, p := range ordered {
		result = append(result, plugindomain.Info{
			ID:             p.Manifest.ID,
			Name:           p.Manifest.Name,
			Version:        p.Manifest.Version,
			Description:    p.Manifest.Description,
			Capabilities:   p.Manifest.Capabilities,
			Settings:       plugindomain.ListSettings(p.Manifest, v.settings(p.Manifest.ID)),
			IsActive:       r.sups.IsActive(p.Manifest.ID),
			Permissions:    p.Manifest.Permissions,
			Failure:        r.failures.Get(p.Manifest.ID, p.Manifest.Name),
			HasFrontend:    p.Manifest.HasFrontend(),
			HasBackend:     p.Manifest.HasBackend(),
			Readers:        r.readers.ListReaderInfos(enabled, p.Manifest),
			SettingsViews:  slices.Clone(p.Manifest.SettingsViews),
			IsOffByDefault: p.Manifest.IsOffByDefault,
		})
	}
	return result
}
