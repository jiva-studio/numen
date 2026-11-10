package settings

import (
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

// Themes provides the themes the window can use: the catalog of themes, and
// the settings file that stores the selection. The catalog offers the themes
// of the enabled plugins of the plugin host, when the config has one.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	files, err := c.openThemeFiles()
	sources := c.getThemeSources(files)
	store := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(c.store()).Execute,
		settingsusecase.NewWriteAppearance(c.writeSettings()).Execute,
		say,
	)
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
