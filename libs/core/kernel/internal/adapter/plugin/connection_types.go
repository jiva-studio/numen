package plugin

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ConnectionPlugins provides the plugins and their manifests. GetPlugin answers only for an enabled plugin.
type ConnectionPlugins interface {
	GetPlugins() []manifest.Discovered
	GetPlugin(id string) (manifest.Discovered, bool)
}

// ConnectionTypes lists the kinds of connection the manifests of the enabled plugins declare. It answers
// from the discovered manifests and starts no plugin process.
type ConnectionTypes struct {
	host ConnectionPlugins
}

// NewConnectionTypes returns the list over the plugins the host registers.
func NewConnectionTypes(plugins ConnectionPlugins) *ConnectionTypes {
	return &ConnectionTypes{host: plugins}
}

// ListTypes returns the types of the enabled plugins, in the order of the plugin ids and then of
// the manifest.
func (r *ConnectionTypes) ListTypes() []connection.Type {
	var types []connection.Type
	for _, discovered := range r.host.GetPlugins() {
		plugin, isEnabled := r.host.GetPlugin(discovered.Manifest.ID)
		if !isEnabled {
			continue
		}
		for _, declared := range plugin.Manifest.Contributes.ConnectionTypes {
			types = append(types, connection.Type{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, PluginVersion: plugin.Manifest.Version,
				IsAvailable: true, Declaration: declared,
			})
		}
	}
	return types
}

// ListOperationTitles returns the titles the enabled plugins give operations, one for each name: the
// first declaration in the order of the plugin ids.
func (r *ConnectionTypes) ListOperationTitles() []manifest.Operation {
	var titles []manifest.Operation
	named := map[string]bool{}
	for _, discovered := range r.host.GetPlugins() {
		plugin, isEnabled := r.host.GetPlugin(discovered.Manifest.ID)
		if !isEnabled {
			continue
		}
		for _, declared := range plugin.Manifest.Contributes.Operations {
			if !named[declared.Name] {
				named[declared.Name] = true
				titles = append(titles, declared)
			}
		}
	}
	return titles
}

// GetType returns the type with the id, and false where no enabled plugin declares it.
func (r *ConnectionTypes) GetType(typeID string) (connection.Type, bool) {
	pluginID, localID, isValid := manifest.SplitConnectionTypeID(typeID)
	if !isValid {
		return connection.Type{}, false
	}
	plugin, isEnabled := r.host.GetPlugin(pluginID)
	if !isEnabled {
		return connection.Type{}, false
	}
	declared, isDeclared := plugin.Manifest.Contributes.GetConnectionType(localID)
	if !isDeclared {
		return connection.Type{}, false
	}
	return connection.Type{
		PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, PluginVersion: plugin.Manifest.Version,
		IsAvailable: true, Declaration: declared,
	}, true
}
