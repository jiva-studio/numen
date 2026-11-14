package pluginport

import (
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// PluginManifests gives the manifest of a plugin by its id.
type PluginManifests interface {
	GetPlugin(id string) (manifest.Discovered, bool)
}
