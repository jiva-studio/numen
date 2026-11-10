package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

// Themes provides the themes the window can use: the catalog of themes, and
// the settings file that stores the selection. The catalog offers the themes
// of the enabled plugins of the plugin host, when the config has one.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	files, err := c.openThemeFiles()
	sources := c.getThemeSources(files)
	store := settingsadapter.NewAppearanceStore(c.readAppearance, c.writeAppearance, say)
	scales := themeusecase.NewScales(c.InterfaceScale, c.TextScale)
	interfaceScale, textScale := settingsadapter.GetInterfaceScaleBounds(), settingsadapter.GetTextScaleBounds()
	return theme.NewService(
		themeusecase.NewGetThemes(sources, store, store, scales,
			themeusecase.Bounds{Min: interfaceScale.Min, Max: interfaceScale.Max},
			themeusecase.Bounds{Min: textScale.Min, Max: textScale.Max}),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, store, store, scales),
		themeusecase.NewWatchThemes(sources, files, 0),
	), err
}

// CheckInterfaceScale returns an error when scale is outside the interface scale bounds.
func CheckInterfaceScale(name string, scale float64) error {
	return settingsadapter.GetInterfaceScaleBounds().Check(name, scale)
}

// CheckTextScale returns an error when scale is outside the text scale bounds.
func CheckTextScale(name string, scale float64) error {
	return settingsadapter.GetTextScaleBounds().Check(name, scale)
}

func (c Config) getThemeSources(files themefiles.Files) themeusecase.Sources {
	sources := themeusecase.Sources{Presets: files, Own: files, Plugins: nil}
	if c.PluginHost != nil {
		sources.Plugins = themebridge.Open(c.PluginHost)
	}
	return sources
}

func (c Config) openThemeFiles() (themefiles.Files, error) {
	if c.ThemesPath != "" {
		return themefiles.OpenAt(c.ThemesPath, c.Watches)
	}
	if folder, selected := c.GetAdjacentPath("themes"); selected {
		return themefiles.OpenAt(folder, c.Watches)
	}
	return themefiles.Open(c.Watches)
}

func (c Config) readAppearance() (themedomain.Appearance, error) {
	path, err := c.SettingsFile()
	if err != nil {
		return themedomain.Appearance{}, err
	}
	activeSettings, err := c.GetSettingsAt(path)
	if err != nil {
		return themedomain.Appearance{}, err
	}
	return themedomain.Appearance{
		ThemeName:      activeSettings.Appearance.Theme,
		Mode:           settingsadapter.Mode(activeSettings.Appearance.Mode),
		InterfaceScale: activeSettings.Appearance.InterfaceScale,
		TextScale:      activeSettings.Appearance.TextScale,
	}, nil
}

func (c Config) writeAppearance(selected themedomain.Appearance) error {
	appearance := func(key string) []string { return []string{settingsadapter.OwnerApp, "appearance", key} }
	values := []struct {
		path  []string
		value any
	}{
		{appearance("theme"), selected.ThemeName},
		{appearance("mode"), settingsadapter.GetModeName(selected.Mode)},
	}
	if selected.InterfaceScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearance("interfaceScale"), selected.InterfaceScale})
	}
	if selected.TextScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearance("textScale"), selected.TextScale})
	}

	writing := make([]settingsdomain.Setting, 0, len(values))
	for _, one := range values {
		encoded, err := json.Marshal(one.value)
		if err != nil {
			return fmt.Errorf("%s: %w", strings.Join(one.path, "."), err)
		}
		writing = append(writing, settingsdomain.Setting{Path: one.path, JSON: string(encoded)})
	}
	return c.WriteSettings(writing)
}
