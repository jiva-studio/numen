// Package connectiontypes lists the kinds of connection the manifests of the enabled plugins
// declare. It answers from the discovered manifests and starts no plugin process.
package connectiontypes

import (
	"github.com/jiva-studio/numen/libs/core/usecase/connection/connectioncontract"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

// Host provides the plugins and their manifests. GetPlugin answers only for an enabled plugin.
type Host interface {
	GetPluginInfos() []plugincontract.PluginInfo
	GetPlugin(id string) (plugincontract.DiscoveredPlugin, bool)
}

// Registry is the types of the enabled plugins the host lists.
type Registry struct {
	host Host
}

// NewRegistry returns the registry over the plugins the host lists.
func NewRegistry(host Host) *Registry {
	return &Registry{host: host}
}

// ListTypes returns the types of the enabled plugins, in the order of the plugin ids and then of
// the manifest.
func (r *Registry) ListTypes() []connectioncontract.Type {
	var types []connectioncontract.Type
	for _, info := range r.host.GetPluginInfos() {
		plugin, isEnabled := r.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, declared := range plugin.Manifest.Contributes.ConnectionTypes {
			types = append(types, connectioncontract.Type{
				PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, IsAvailable: true, Declaration: declared,
			})
		}
	}
	return types
}

// GetType returns the type with the id, and false where no enabled plugin declares it.
func (r *Registry) GetType(typeID string) (connectioncontract.Type, bool) {
	pluginID, localID, isValid := manifest.SplitConnectionTypeID(typeID)
	if !isValid {
		return connectioncontract.Type{}, false
	}
	plugin, isEnabled := r.host.GetPlugin(pluginID)
	if !isEnabled {
		return connectioncontract.Type{}, false
	}
	declared, isDeclared := plugin.Manifest.Contributes.GetConnectionType(localID)
	if !isDeclared {
		return connectioncontract.Type{}, false
	}
	return connectioncontract.Type{
		PluginID: plugin.Manifest.ID, PluginName: plugin.Manifest.Name, IsAvailable: true, Declaration: declared,
	}, true
}
