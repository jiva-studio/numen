package container

import (
	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
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

const appearancePrefix = "appearance"

func (c Config) writeAppearance(selectedVal theme.Appearance, scales *theme.Scales) error {
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	writing := []settings.Setting{
		{At: []string{appearancePrefix, "theme"}, Written: selectedVal.ThemeName},
		{At: []string{appearancePrefix, "mode"}, Written: settings.GetModeName(selectedVal.Mode)},
	}
	if selectedVal.InterfaceScale > 0 {
		err := settings.GetInterfaceScaleBounds().Check("appearance.interface_scale", selectedVal.InterfaceScale)
		if err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{appearancePrefix, "interface_scale"}, Written: selectedVal.InterfaceScale})
	}
	if selectedVal.TextScale > 0 {
		if err := settings.GetTextScaleBounds().Check("appearance.text_scale", selectedVal.TextScale); err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{appearancePrefix, "text_scale"}, Written: selectedVal.TextScale})
	}
	into := DefaultSettings()
	if err := c.getResources().settingsFiles.Save(path, &into, writing...); err != nil {
		return err
	}
	scales.ClearSizes(selectedVal)
	return nil
}
