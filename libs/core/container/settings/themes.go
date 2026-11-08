package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	settingsadapter "github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/port/settingsport"
)

// Themes provides the themes the window can use: the catalog of themes, and
// the settings file that stores the selection. The catalog offers the themes
// of the enabled plugins of the plugin host, when the config has one.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	catalog, err := c.getCatalogue()
	scales := theme.NewScales(c.InterfaceScale, c.TextScale)
	return &theme.Service{
		Catalog: catalog,
		Settings: theme.NewAppearances(
			func() (theme.Appearance, error) { return c.readAppearance(scales) },
			func(choice theme.Appearance) error { return c.writeAppearance(choice, scales) },
			say,
		),
		InterfaceScaleBounds: theme.Bounds{
			Min: settingsadapter.GetInterfaceScaleBounds().Min, Max: settingsadapter.GetInterfaceScaleBounds().Max,
		},
		TextScaleBounds: theme.Bounds{
			Min: settingsadapter.GetTextScaleBounds().Min, Max: settingsadapter.GetTextScaleBounds().Max,
		},
	}, err
}

// CheckInterfaceScale returns an error when scale is outside the interface scale bounds.
func CheckInterfaceScale(name string, scale float64) error {
	return settingsadapter.GetInterfaceScaleBounds().Check(name, scale)
}

// CheckTextScale returns an error when scale is outside the text scale bounds.
func CheckTextScale(name string, scale float64) error {
	return settingsadapter.GetTextScaleBounds().Check(name, scale)
}

func (c Config) getCatalogue() (theme.Catalog, error) {
	catalog, err := c.openCatalogue()
	if c.PluginHost != nil {
		catalog = catalog.WithPlugins(themebridge.Open(c.PluginHost))
	}
	return catalog, err
}

func (c Config) openCatalogue() (theme.Catalog, error) {
	if c.ThemesPath != "" {
		return theme.OpenAt(c.ThemesPath, c.Watches)
	}
	if folder, selected := c.GetAdjacentPath("themes"); selected {
		return theme.OpenAt(folder, c.Watches)
	}
	return theme.Open(c.Watches)
}

func (c Config) readAppearance(scales *theme.Scales) (theme.Appearance, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return theme.Appearance{}, err
	}
	activeSettings, err := c.GetSettingsAt(path)
	if err != nil {
		return theme.Appearance{}, err
	}
	current := theme.Appearance{
		ThemeName:      activeSettings.Appearance.Theme,
		Mode:           settingsadapter.Mode(activeSettings.Appearance.Mode),
		InterfaceScale: activeSettings.Appearance.InterfaceScale,
		TextScale:      activeSettings.Appearance.TextScale,
	}
	scales.Apply(&current)
	return current, nil
}

func (c Config) writeAppearance(selectedVal theme.Appearance, scales *theme.Scales) error {
	appearance := func(key string) []string { return []string{settingsadapter.OwnerApp, "appearance", key} }
	values := []struct {
		path  []string
		value any
	}{
		{appearance("theme"), selectedVal.ThemeName},
		{appearance("mode"), settingsadapter.GetModeName(selectedVal.Mode)},
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
	if err := c.WriteSettings(writing); err != nil {
		return err
	}
	scales.ClearSizes(selectedVal)
	return nil
}
