package plugin

import (
	"slices"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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

// Registry answers which plugins the host knows and which of them are enabled. In safe mode it
// knows none.
type Registry struct {
	t            *table
	failures     *Failures
	contentTypes *ContentTypes
}

func newRegistry(t *table, failures *Failures, contentTypes *ContentTypes) *Registry {
	return &Registry{t: t, failures: failures, contentTypes: contentTypes}
}

// GetPlugins returns all discovered plugins registered with the host, in the order of their ids.
func (r *Registry) GetPlugins() []manifest.Discovered {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	if r.t.safeMode {
		return nil
	}

	return r.t.getPluginsInOrderLocked()
}

// GetPlugin retrieves a single enabled plugin by its unique identifier.
func (r *Registry) GetPlugin(id string) (manifest.Discovered, bool) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	if r.t.safeMode {
		return manifest.Discovered{}, false
	}

	p, ok := r.t.plugins[id]
	if !ok || !r.t.isPluginEnabledLocked(id) {
		return manifest.Discovered{}, false
	}
	return p, true
}

// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
func (r *Registry) IsPluginEnabled(pluginID string) bool {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()
	return r.t.isPluginEnabledLocked(pluginID)
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *plugin.UnavailableError: Disabled when a
// plugin declares it and it is switched off or the host is in safe mode, NotInstalled when
// none declares it.
func (r *Registry) ResolveCapability(capability string) (manifest.Discovered, error) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	reason := plugindomain.NotInstalled
	for _, p := range r.t.getPluginsInOrderLocked() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if r.t.safeMode || !r.t.isPluginEnabledLocked(p.Manifest.ID) {
			reason = plugindomain.Disabled
			continue
		}
		return p, nil
	}
	missing := &plugindomain.UnavailableError{Capability: capability, Reason: reason}
	if r.t.safeMode {
		missing.Err = ErrSafeModeActive
	}
	return manifest.Discovered{}, missing
}

// ResolveService finds the enabled plugin with the lowest id among those that serve the service to
// the windows.
func (r *Registry) ResolveService(service string) (manifest.Discovered, bool) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	if r.t.safeMode {
		return manifest.Discovered{}, false
	}
	for _, p := range r.t.getPluginsInOrderLocked() {
		if p.Manifest.HasService(service) && r.t.isPluginEnabledLocked(p.Manifest.ID) {
			return p, true
		}
	}
	return manifest.Discovered{}, false
}

// ResolveNoteProfile returns how the index stores the notes of a type, as
// defined by the enabled plugin with the lowest id among those declaring the
// type.
func (r *Registry) ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	if r.t.safeMode {
		return domain.NoteProfile{}, false
	}
	for _, p := range r.t.getPluginsInOrderLocked() {
		if declared, found := p.Manifest.GetNoteType(string(noteType)); found && r.t.isPluginEnabledLocked(p.Manifest.ID) {
			return declared.GetProfile(), true
		}
	}
	return domain.NoteProfile{}, false
}

// FindPluginByCapability finds the plugin ResolveCapability finds, and whether there is one.
func (r *Registry) FindPluginByCapability(capability string) (manifest.Discovered, bool) {
	p, err := r.ResolveCapability(capability)
	return p, err == nil
}

// GetPluginSettingDeclarations returns the settings the manifest of the plugin declares, and false
// for a plugin the host does not know.
func (r *Registry) GetPluginSettingDeclarations(pluginID string) ([]manifest.Setting, bool) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()
	p, ok := r.t.plugins[pluginID]
	if !ok {
		return nil, false
	}
	return slices.Clone(p.Manifest.Settings), true
}

// GetPluginStatus returns the status of the specified plugin.
func (r *Registry) GetPluginStatus(pluginID string) (PluginStatus, bool) {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	p, ok := r.t.plugins[pluginID]
	if !ok {
		return PluginStatus{}, false
	}
	return r.getPluginStatusLocked(p.Manifest.ID), true
}

func (r *Registry) getPluginStatusLocked(pluginID string) PluginStatus {
	status := PluginStatus{
		ID:            pluginID,
		Enabled:       r.t.isPluginEnabledLocked(pluginID),
		IsActive:      r.t.isPluginActiveLocked(pluginID),
		Failed:        false,
		FailureReason: "",
		Dir:           "",
	}
	if sup, ok := r.t.supervisors[pluginID]; ok {
		if cause := sup.GetActivationFailure(); cause != nil {
			status.Failed = true
			status.FailureReason = cause.Error()
		}
	}
	return status
}

// GetPluginStatuses returns statuses for all registered plugins, in the order of their ids.
func (r *Registry) GetPluginStatuses() []PluginStatus {
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	statuses := make([]PluginStatus, 0, len(r.t.plugins)+len(r.t.problems))
	for _, p := range r.t.getPluginsInOrderLocked() {
		statuses = append(statuses, r.getPluginStatusLocked(p.Manifest.ID))
	}
	for _, problem := range r.t.problems {
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
	r.t.mu.RLock()
	defer r.t.mu.RUnlock()

	result := make([]plugindomain.Info, 0, len(r.t.plugins))
	for _, p := range r.t.getPluginsInOrderLocked() {
		result = append(result, plugindomain.Info{
			ID:            p.Manifest.ID,
			Name:          p.Manifest.Name,
			Version:       p.Manifest.Version,
			Description:   p.Manifest.Description,
			Capabilities:  p.Manifest.Capabilities,
			Settings:      plugindomain.ListSettings(p.Manifest, r.t.getPluginSettingsLocked(p.Manifest.ID)),
			IsActive:      r.t.isPluginActiveLocked(p.Manifest.ID),
			Permissions:   p.Manifest.Permissions,
			Secrets:       p.Manifest.Secrets,
			Failure:       r.failures.getFailureLocked(p.Manifest.ID),
			HasFrontend:   p.Manifest.HasFrontend(),
			HasBackend:    p.Manifest.HasBackend(),
			Readers:       r.contentTypes.listReaderInfosLocked(p.Manifest),
			SettingsViews: slices.Clone(p.Manifest.SettingsViews),
		})
	}
	return result
}
