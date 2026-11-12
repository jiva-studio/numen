// Package theme lists, reads, applies and watches the themes that restyle the window.
//
// A theme is one CSS file redeclaring the tokens the interface renders with. The default theme
// ships inside the binary, enabled plugins contribute themes through their manifests, and the rest
// are files in the user's themes folder; all form one list with one format, and one name applies
// any of them. Nothing here reads inside a theme beyond whether it pins `color-scheme`.
package theme

import (
	"errors"
	"fmt"
	"strings"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/themeport"
)

// Sources is where the themes are read from. Plugins may be nil, for a build with no plugin host.
type Sources struct {
	Presets themeport.PresetThemes
	Own     themeport.OwnThemes
	Plugins themeport.PluginThemes
}

// catalog is every theme this installation offers. A folder that could not be read still leaves the
// default theme, so the window opens with a theme regardless of disk state.
type catalog struct {
	sources Sources
}

func (c catalog) list() []themedomain.Theme {
	var themes []themedomain.Theme
	for _, file := range c.sources.Presets.ListPresets() {
		themes = append(themes, describe(themedomain.Preset, file))
	}
	themes = append(themes, c.listPluginThemes()...)
	for _, file := range c.sources.Own.ListOwnThemes() {
		themes = append(themes, describe(themedomain.Mine, file))
	}
	return themes
}

func describe(origin themedomain.Origin, file themedomain.File) themedomain.Theme {
	return themedomain.Theme{
		Name:       themedomain.GetName(origin, file.Title),
		Title:      file.Title,
		Origin:     origin,
		IsPinned:   file.IsPinned,
		PluginID:   "",
		Appearance: "",
	}
}

func (c catalog) listPluginThemes() []themedomain.Theme {
	if c.sources.Plugins == nil {
		return nil
	}
	declared := c.sources.Plugins.ListThemes()
	themes := make([]themedomain.Theme, 0, len(declared))
	for _, one := range declared {
		themes = append(themes, themedomain.Theme{
			Name:       themedomain.PluginName(one.PluginID, one.ID),
			Title:      one.Name,
			Origin:     themedomain.Plugin,
			IsPinned:   one.Appearance != "",
			PluginID:   one.PluginID,
			Appearance: one.Appearance,
		})
	}
	return themes
}

func (c catalog) listPluginThemeNames() []string {
	themes := c.listPluginThemes()
	names := make([]string, 0, len(themes))
	for _, one := range themes {
		names = append(names, one.Name)
	}
	return names
}

func (c catalog) read(name string) (string, error) {
	if strings.HasPrefix(name, themedomain.PluginPrefix) {
		return c.readPluginTheme(name)
	}
	origin, title, err := themedomain.SplitName(name)
	if err != nil {
		return "", err
	}
	if origin == themedomain.Preset {
		text, err := c.sources.Presets.ReadPreset(title)
		if err != nil {
			return "", fmt.Errorf("%s: %w", name, themedomain.ErrNoTheme)
		}
		return text, nil
	}
	text, err := c.sources.Own.ReadOwnTheme(title)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

func (c catalog) readPluginTheme(name string) (string, error) {
	pluginID, themeID, isValid := themedomain.SplitPluginName(name)
	if !isValid || c.sources.Plugins == nil {
		return "", fmt.Errorf("%s: %w", name, themedomain.ErrNoTheme)
	}
	text, err := c.sources.Plugins.ReadTheme(pluginID, themeID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return text, nil
}

func (c catalog) getAppliedTheme(name string) (applied, missing string) {
	if name == "" {
		return themedomain.Default, ""
	}
	if _, err := c.read(name); err != nil {
		return themedomain.Default, name
	}
	return name, ""
}

func isMissingTheme(err error) bool {
	return errors.Is(err, themedomain.ErrNoTheme) || errors.Is(err, themedomain.ErrNoFolder)
}
