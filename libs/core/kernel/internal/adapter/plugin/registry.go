package plugin

import (
	"errors"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/contenttypes"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

var (
	errPluginNotFound = errors.New("plugin not found")
	// ErrSafeModeActive is what a call fails with while the host runs in safe mode.
	ErrSafeModeActive = errors.New("plugin host is running in safe mode")
	errHostClosed     = errors.New("plugin host is shut down")
)

// Registry answers which plugins the host knows and which of them are enabled. In safe mode it
// knows none.
type Registry struct {
	set      *pluginSet
	mode     safeModeReader
	sups     *supervision.Supervisors
	failures *supervision.Failures
	readers  *contenttypes.Lookup
}

// safeModeReader says whether the host runs in safe mode.
type safeModeReader interface {
	IsSafeMode() bool
}

func newRegistry(
	set *pluginSet, mode safeModeReader, sups *supervision.Supervisors, failures *supervision.Failures, readers *contenttypes.Lookup,
) *Registry {
	return &Registry{set: set, mode: mode, sups: sups, failures: failures, readers: readers}
}

// GetPlugins returns all discovered plugins registered with the host, in the order of their ids.
func (r *Registry) GetPlugins() []manifest.Discovered {
	v, release := r.set.view()
	defer release()
	if r.mode.IsSafeMode() {
		return nil
	}
	return v.ordered()
}

// GetPlugin retrieves a single enabled plugin by its unique identifier.
func (r *Registry) GetPlugin(id string) (manifest.Discovered, bool) {
	v, release := r.set.view()
	defer release()
	if r.mode.IsSafeMode() {
		return manifest.Discovered{}, false
	}
	p, ok := v.get(id)
	if !ok || !v.isEnabled(id) {
		return manifest.Discovered{}, false
	}
	return p, true
}

// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
func (r *Registry) IsPluginEnabled(pluginID string) bool {
	v, release := r.set.view()
	defer release()
	return v.isEnabled(pluginID)
}

// ResolveCapability finds the enabled plugin with the lowest id among those declaring the given
// capability. Where none is enabled, the error is a *plugin.UnavailableError: Disabled when a
// plugin declares it and it is switched off or the host is in safe mode, NotInstalled when
// none declares it.
func (r *Registry) ResolveCapability(capability string) (manifest.Discovered, error) {
	v, release := r.set.view()
	defer release()

	isSafe := r.mode.IsSafeMode()
	reason := plugindomain.NotInstalled
	for _, p := range v.ordered() {
		if !p.Manifest.HasCapability(capability) {
			continue
		}
		if isSafe || !v.isEnabled(p.Manifest.ID) {
			reason = plugindomain.Disabled
			continue
		}
		return p, nil
	}
	missing := &plugindomain.UnavailableError{Capability: capability, Reason: reason}
	if isSafe {
		missing.Err = ErrSafeModeActive
	}
	return manifest.Discovered{}, missing
}

// ResolveService finds the enabled plugin with the lowest id among those that serve the service to
// the windows.
func (r *Registry) ResolveService(service string) (manifest.Discovered, bool) {
	v, release := r.set.view()
	defer release()
	for _, p := range v.enabled() {
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
	v, release := r.set.view()
	defer release()
	for _, p := range v.enabled() {
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
	p, ok := r.set.getKnown(pluginID)
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
