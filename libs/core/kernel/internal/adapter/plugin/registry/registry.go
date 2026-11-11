// Package registry answers which plugins the host knows, which of them are enabled, what state they are in, and which
// type a file has over them.
package registry

import (
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostmode"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/contenttypes"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// modeState is what the registry reads of the mode of the host.
type modeState interface {
	IsSafeMode() bool
}

// Registry answers which plugins the host knows and which of them are enabled. In safe mode it
// knows none.
type Registry struct {
	set      *pluginset.Set
	mode     modeState
	sups     *supervision.Supervisors
	failures *supervision.Failures
	readers  *contenttypes.Lookup
}

// New returns the registry over the set of plugins, the mode of the host, the supervisors, the failures and the readers.
func New(
	set *pluginset.Set, mode modeState, sups *supervision.Supervisors, failures *supervision.Failures, readers *contenttypes.Lookup,
) *Registry {
	return &Registry{set: set, mode: mode, sups: sups, failures: failures, readers: readers}
}

// GetPlugins returns all discovered plugins registered with the host, in the order of their ids.
func (r *Registry) GetPlugins() []manifest.Discovered {
	v, release := r.set.View()
	defer release()
	if r.mode.IsSafeMode() {
		return nil
	}
	return v.Ordered()
}

// GetPlugin retrieves a single enabled plugin by its unique identifier.
func (r *Registry) GetPlugin(id string) (manifest.Discovered, bool) {
	v, release := r.set.View()
	defer release()
	if r.mode.IsSafeMode() {
		return manifest.Discovered{}, false
	}
	p, ok := v.Get(id)
	if !ok || !v.IsEnabled(id) {
		return manifest.Discovered{}, false
	}
	return p, true
}

// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
func (r *Registry) IsPluginEnabled(pluginID string) bool {
	v, release := r.set.View()
	defer release()
	return v.IsEnabled(pluginID)
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *plugin.UnavailableError: Disabled when a
// plugin declares it and it is switched off or the host is in safe mode, NotInstalled when
// none declares it.
func (r *Registry) ResolveCapability(capability string) (manifest.Discovered, error) {
	v, release := r.set.View()
	defer release()

	isSafe := r.mode.IsSafeMode()
	reason := plugindomain.NotInstalled
	for _, p := range v.Ordered() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if isSafe || !v.IsEnabled(p.Manifest.ID) {
			reason = plugindomain.Disabled
			continue
		}
		return p, nil
	}
	missing := &plugindomain.UnavailableError{Capability: capability, Reason: reason}
	if isSafe {
		missing.Err = hostmode.ErrSafeModeActive
	}
	return manifest.Discovered{}, missing
}

// ResolveService finds the enabled plugin with the lowest id among those that serve the service to
// the windows.
func (r *Registry) ResolveService(service string) (manifest.Discovered, bool) {
	v, release := r.set.View()
	defer release()
	for _, p := range v.Enabled() {
		if p.Manifest.HasService(service) {
			return p, true
		}
	}
	return manifest.Discovered{}, false
}

// ResolveNoteProfile returns how the index stores the notes of a type, as
// defined by the enabled plugin with the lowest id among those declaring the
// type.
func (r *Registry) ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool) {
	v, release := r.set.View()
	defer release()
	for _, p := range v.Enabled() {
		if declared, found := p.Manifest.GetNoteType(string(noteType)); found {
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
	p, ok := r.set.GetKnown(pluginID)
	if !ok {
		return nil, false
	}
	return slices.Clone(p.Manifest.Settings), true
}

// IsEnabled reports whether the host knows the plugin and has it enabled. In safe mode it has none.
func (r *Registry) IsEnabled(pluginID string) bool {
	_, ok := r.GetPlugin(pluginID)
	return ok
}

// Set returns the set of plugins the registry answers from.
func (r *Registry) Set() *pluginset.Set {
	return r.set
}

// Supervisors returns the supervisors whose states the registry reports.
func (r *Registry) Supervisors() *supervision.Supervisors {
	return r.sups
}
