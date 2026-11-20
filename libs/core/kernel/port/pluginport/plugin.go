package pluginport

import (
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// PluginResolver locates plugins by capability or identifier. Of several enabled plugins that
// declare a capability, the one with the lowest id is found.
type PluginResolver interface {
	// ResolveCapability returns the plugin for the capability. Where none answers, the error is an
	// *UnavailableError saying whether none is installed or every one is disabled.
	ResolveCapability(capability string) (manifest.Discovered, error)
	ServiceResolver
	// ResolveNoteProfile returns how the index keeps the notes of a type, as the enabled plugin that
	// declares the type defines, and false where none does.
	ResolveNoteProfile(noteType domain.NoteType) (domain.NoteProfile, bool)
	GetPlugin(id string) (manifest.Discovered, bool)
}

// ServiceResolver finds the plugin that serves a service to the windows.
type ServiceResolver interface {
	// ResolveService returns the enabled plugin with the lowest id among those that serve the service
	// to the windows, and false where none does.
	ResolveService(service string) (manifest.Discovered, bool)
}

// PluginEnablement reports whether the settings of a plugin leave it enabled.
type PluginEnablement interface {
	// IsPluginEnabled reports whether the settings of the plugin leave it enabled.
	IsPluginEnabled(pluginID string) bool
}

// ConfigWatcher tells when a plugin has been given its configuration.
type ConfigWatcher interface {
	WatchPluginConfig(pluginID string, fn func()) (stop func())
}
