// Package themebridge lists and reads the themes the manifests of the enabled plugins declare. It
// answers from the discovered manifests and starts no plugin process.
package themebridge

import (
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugintheme/themecontract"
)

// Host provides the plugins and their manifests.
type Host interface {
	plugincontract.PluginDiscoverer
	plugincontract.PluginResolver
}

// Themes is the themes of the plugins the host lists.
type Themes struct {
	host Host
}

var _ themecontract.PluginThemes = Themes{host: nil}

// Open makes the bridge to the plugins the host lists.
func Open(host Host) Themes {
	return Themes{host: host}
}

// ListThemes returns the themes of the enabled plugins, in the order of the plugin ids.
func (t Themes) ListThemes() []theme.PluginTheme {
	var themes []theme.PluginTheme
	for _, info := range t.host.GetPluginInfos() {
		plugin, isEnabled := t.host.GetPlugin(info.ID)
		if !isEnabled {
			continue
		}
		for _, declared := range plugin.Manifest.Contributes.Themes {
			themes = append(themes, theme.PluginTheme{
				PluginID:   plugin.Manifest.ID,
				ID:         declared.ID,
				Name:       declared.Name,
				Appearance: declared.Appearance,
			})
		}
	}
	return themes
}

// ReadTheme returns the stylesheet of a theme of an enabled plugin. The file is read inside the plugin
// folder, through no link that leaves it.
func (t Themes) ReadTheme(pluginID, themeID string) (string, error) {
	plugin, isEnabled := t.host.GetPlugin(pluginID)
	if !isEnabled {
		return "", theme.ErrNoTheme
	}
	for _, declared := range plugin.Manifest.Contributes.Themes {
		if declared.ID != themeID {
			continue
		}
		text, err := manifestjson.ReadThemeFile(plugin.Dir, declared.File)
		if errors.Is(err, fs.ErrNotExist) {
			return "", theme.ErrNoTheme
		}
		return text, err
	}
	return "", theme.ErrNoTheme
}
