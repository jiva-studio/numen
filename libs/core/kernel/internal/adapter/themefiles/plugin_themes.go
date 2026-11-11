package themefiles

import (
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// PluginHost provides the plugins and their manifests.
type PluginHost interface {
	GetPluginInfos() []plugin.Info
	GetPlugin(id string) (manifest.Discovered, bool)
}

// PluginThemes lists and reads the themes the manifests of the enabled plugins declare. It answers from the
// discovered manifests and starts no plugin process.
type PluginThemes struct {
	host PluginHost
}

// NewPluginThemes returns the themes of the plugins the host lists.
func NewPluginThemes(host PluginHost) PluginThemes {
	return PluginThemes{host: host}
}

// ListThemes returns the themes of the enabled plugins, in the order of the plugin ids.
func (t PluginThemes) ListThemes() []theme.PluginTheme {
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
func (t PluginThemes) ReadTheme(pluginID, themeID string) (string, error) {
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
