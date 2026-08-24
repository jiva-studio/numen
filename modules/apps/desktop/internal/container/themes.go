package container

import (
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/theme"
)

// Themes is what a person may dress the window in: the catalogue of themes, and
// the settings file a choice out of it is written into.
//
// An error here is the themes folder — a machine that names no configuration
// folder, a folder that could not be made — and what the binary ships is
// offered either way. Whatever a person is told is told through say.
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	catalogue, err := c.catalogue()
	return &theme.Service{
		Catalogue: catalogue,
		Say:       say,
		Dressed:   c.dressed,
		Wear:      c.wear,
	}, err
}

func (c Config) catalogue() (theme.Catalogue, error) {
	if c.ThemesPath != "" {
		return theme.At(c.ThemesPath)
	}
	if folder, chosen := c.beside("themes"); chosen {
		return theme.At(folder)
	}
	return theme.Open()
}

// dressed and wear are the settings file as the themes need it: one section of
// it read, and two fields of it written.
func (c Config) dressed() (theme.Dress, error) {
	path, err := c.settingsFile()
	if err != nil {
		return theme.Dress{}, err
	}
	said, err := settings.At(path)
	if err != nil {
		return theme.Dress{}, err
	}
	return theme.Dress{Theme: said.Appearance.Theme, Mode: mode(said.Appearance.Mode)}, nil
}

func (c Config) wear(worn theme.Dress) error {
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	return settings.Save(path,
		settings.Setting{At: []string{"appearance", "theme"}, Value: worn.Theme},
		settings.Setting{At: []string{"appearance", "mode"}, Value: word(worn.Mode)},
	)
}

// mode and word are the settings' word for a mode and the schema's value for
// it, put side by side in the one place that knows both.
func mode(said string) v1.Mode {
	switch said {
	case settings.ModeLight:
		return v1.Mode_MODE_LIGHT
	case settings.ModeDark:
		return v1.Mode_MODE_DARK
	}
	return v1.Mode_MODE_SYSTEM
}

func word(mode v1.Mode) string {
	switch mode {
	case v1.Mode_MODE_LIGHT:
		return settings.ModeLight
	case v1.Mode_MODE_DARK:
		return settings.ModeDark
	}
	return settings.ModeSystem
}
