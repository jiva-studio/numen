package container

import (
	"sync"

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
			cfg:    c,
			scales: &scales{uiScale: c.InterfaceScale, textScale: c.TextScale},
			say:    say,
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
	cfg    Config
	scales *scales
	say    func(string)
}

func (a appearances) Read() (theme.Appearance, error) { return a.cfg.readAppearance(a.scales) }

func (a appearances) Write(chosen theme.Appearance) error { return a.cfg.wear(chosen, a.scales) }

func (a appearances) Warn(why string) {
	if a.say != nil {
		a.say(why)
	}
}

// scales holds CLI scale overrides until explicitly changed.
type scales struct {
	mu                 sync.Mutex
	uiScale, textScale float64
}

// apply overrides appearance scales with CLI flags if set.
func (l *scales) apply(worn *theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.uiScale > 0 {
		worn.InterfaceScale = l.uiScale
	}
	if l.textScale > 0 {
		worn.TextScale = l.textScale
	}
}

// clearSizes clears CLI scale overrides when the user configures explicit values.
func (l *scales) clearSizes(chosen theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if chosen.InterfaceScale > 0 {
		l.uiScale = 0
	}
	if chosen.TextScale > 0 {
		l.textScale = 0
	}
}

func (c Config) catalogue() (theme.Catalogue, error) {
	if c.ThemesPath != "" {
		return theme.OpenAt(c.ThemesPath)
	}
	if folder, chosen := c.getPathBeside("themes"); chosen {
		return theme.OpenAt(folder)
	}
	return theme.Open()
}

// readAppearance and wear are the settings file as the themes need it: one
// section of it read, and up to four fields of it written.
func (c Config) readAppearance(scales *scales) (theme.Appearance, error) {
	path, err := c.settingsFile()
	if err != nil {
		return theme.Appearance{}, err
	}
	activeSettings, err := c.getSettingsAt(path)
	if err != nil {
		return theme.Appearance{}, err
	}
	worn := theme.Appearance{
		ThemeName:      activeSettings.Appearance.Theme,
		Mode:           settings.Mode(activeSettings.Appearance.Mode),
		InterfaceScale: activeSettings.Appearance.InterfaceScale,
		TextScale:      activeSettings.Appearance.TextScale,
	}
	scales.apply(&worn)
	return worn, nil
}

// wear writes a choice into the file. Both sizes are checked before any of it
// is written, so a number outside what its setting goes to leaves the file as
// it stands.
func (c Config) wear(chosen theme.Appearance, scales *scales) error {
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	writing := []settings.Setting{
		{At: []string{"appearance", "theme"}, Written: chosen.ThemeName},
		{At: []string{"appearance", "mode"}, Written: settings.Word(chosen.Mode)},
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
	into := DefaultSettings()
	if err := settings.Save(path, &into, writing...); err != nil {
		return err
	}
	scales.clearSizes(chosen)
	return nil
}
