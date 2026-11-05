package container

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// Themes provides the themes the window can use: the catalogue of themes, and
// the settings file that stores the selection. The catalogue offers the themes
// of the enabled plugins of the plugin host, when the config has one.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	catalogue, err := c.getCatalogue()
	scales := theme.NewScales(c.InterfaceScale, c.TextScale)
	return &theme.Service{
		Catalogue: catalogue,
		Settings: theme.NewAppearances(
			func() (theme.Appearance, error) { return c.readAppearance(scales) },
			func(choice theme.Appearance) error { return c.writeAppearance(choice, scales) },
			say,
		),
		InterfaceScaleBounds: theme.Bounds{
			Min: settings.GetInterfaceScaleBounds().Min, Max: settings.GetInterfaceScaleBounds().Max,
		},
		TextScaleBounds: theme.Bounds{
			Min: settings.GetTextScaleBounds().Min, Max: settings.GetTextScaleBounds().Max,
		},
	}, err
}

// CheckInterfaceScale is an error naming the scale where the interface scale setting does not take
// it.
func CheckInterfaceScale(name string, scale float64) error {
	return settings.GetInterfaceScaleBounds().Check(name, scale)
}

// CheckTextScale is an error naming the scale where the text scale setting does not take it.
func CheckTextScale(name string, scale float64) error {
	return settings.GetTextScaleBounds().Check(name, scale)
}

func (c Config) getCatalogue() (theme.Catalogue, error) {
	catalogue, err := c.openCatalogue()
	if c.PluginHost != nil {
		catalogue = catalogue.WithPlugins(themebridge.Open(c.PluginHost))
	}
	return catalogue, err
}

func (c Config) openCatalogue() (theme.Catalogue, error) {
	if c.ThemesPath != "" {
		return theme.OpenAt(c.ThemesPath, c.getResources().watches)
	}
	if folder, selected := c.getAdjacentPath("themes"); selected {
		return theme.OpenAt(folder, c.getResources().watches)
	}
	return theme.Open(c.getResources().watches)
}

func (c Config) readAppearance(scales *theme.Scales) (theme.Appearance, error) {
	path, err := c.settingsFile()
	if err != nil {
		return theme.Appearance{}, err
	}
	activeSettings, err := c.getSettingsAt(path)
	if err != nil {
		return theme.Appearance{}, err
	}
	current := theme.Appearance{
		ThemeName:      activeSettings.Appearance.Theme,
		Mode:           settings.Mode(activeSettings.Appearance.Mode),
		InterfaceScale: activeSettings.Appearance.InterfaceScale,
		TextScale:      activeSettings.Appearance.TextScale,
	}
	scales.Apply(&current)
	return current, nil
}

func (c Config) writeAppearance(selectedVal theme.Appearance, scales *theme.Scales) error {
	appearance := func(key string) []string { return []string{settings.OwnerApp, "appearance", key} }
	values := []struct {
		path  []string
		value any
	}{
		{appearance("theme"), selectedVal.ThemeName},
		{appearance("mode"), settings.GetModeName(selectedVal.Mode)},
	}
	if selectedVal.InterfaceScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearance("interfaceScale"), selectedVal.InterfaceScale})
	}
	if selectedVal.TextScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearance("textScale"), selectedVal.TextScale})
	}
	writing := make([]settingsport.Setting, 0, len(values))
	for _, one := range values {
		encoded, err := json.Marshal(one.value)
		if err != nil {
			return fmt.Errorf("%s: %w", strings.Join(one.path, "."), err)
		}
		writing = append(writing, settingsport.Setting{Path: one.path, JSON: string(encoded)})
	}
	if err := c.writeSettings(writing); err != nil {
		return err
	}
	scales.ClearSizes(selectedVal)
	return nil
}
