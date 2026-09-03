package container

import (
	"sync"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
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
		Settings: appearances{
			cfg:  c,
			said: &scales{drawn: c.InterfaceScale, set: c.TextScale},
			say:  say,
		},
		InterfaceScaleBounds: theme.Bounds{
			Least: settings.InterfaceScaleBounds.Least, Most: settings.InterfaceScaleBounds.Most,
		},
		TextScaleBounds: theme.Bounds{
			Least: settings.TextScaleBounds.Least, Most: settings.TextScaleBounds.Most,
		},
	}, err
}

// appearances is the settings file as the themes reach it, with what the
// command line said about size standing over what the file holds.
type appearances struct {
	cfg  Config
	said *scales
	say  func(string)
}

func (a appearances) Read() (theme.Appearance, error) { return a.cfg.dressed(a.said) }

func (a appearances) Write(chosen theme.Appearance) error { return a.cfg.wear(chosen, a.said) }

func (a appearances) Warn(why string) {
	if a.say != nil {
		a.say(why)
	}
}

// scales is what the command line said about size. Each stands over the file
// until a person chooses that size themselves, and zero is not said.
type scales struct {
	mu         sync.Mutex
	drawn, set float64
}

// over puts what was said this launch over what the file holds.
func (l *scales) over(worn *theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.drawn > 0 {
		worn.InterfaceScale = l.drawn
	}
	if l.set > 0 {
		worn.TextScale = l.set
	}
}

// chose lets go of what was said this launch about a size a person has now
// chosen for themselves.
func (l *scales) chose(chosen theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if chosen.InterfaceScale > 0 {
		l.drawn = 0
	}
	if chosen.TextScale > 0 {
		l.set = 0
	}
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
// it read, and up to four fields of it written.
func (c Config) dressed(said *scales) (theme.Appearance, error) {
	path, err := c.settingsFile()
	if err != nil {
		return theme.Appearance{}, err
	}
	held, err := settings.At(path)
	if err != nil {
		return theme.Appearance{}, err
	}
	worn := theme.Appearance{
		ThemeName:      held.Appearance.Theme,
		Mode:           mode(held.Appearance.Mode),
		InterfaceScale: held.Appearance.InterfaceScale,
		TextScale:      held.Appearance.TextScale,
	}
	said.over(&worn)
	return worn, nil
}

// wear writes a choice into the file. Both sizes are checked before any of it
// is written, so a number outside what its setting goes to leaves the file as
// it stands.
func (c Config) wear(chosen theme.Appearance, said *scales) error {
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	writing := []settings.Setting{
		{At: []string{"appearance", "theme"}, Written: chosen.ThemeName},
		{At: []string{"appearance", "mode"}, Written: word(chosen.Mode)},
	}
	if chosen.InterfaceScale > 0 {
		err := settings.InterfaceScaleBounds.Check("appearance.interface_scale", chosen.InterfaceScale)
		if err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{"appearance", "interface_scale"}, Written: chosen.InterfaceScale})
	}
	if chosen.TextScale > 0 {
		if err := settings.TextScaleBounds.Check("appearance.text_scale", chosen.TextScale); err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{"appearance", "text_scale"}, Written: chosen.TextScale})
	}
	if err := settings.Save(path, writing...); err != nil {
		return err
	}
	said.chose(chosen)
	return nil
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
