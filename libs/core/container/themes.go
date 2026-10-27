package container

import (
	"sync"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
)

// Themes is what a person may dress the window in: the catalogue of themes, and
// the settings file a choice out of it is written into.
//
// An error here is the themes folder — a machine that names no configuration
// folder, a folder that could not be made — and what the binary ships is
func (c Config) Themes(say func(string)) (*theme.Service, error) {
	catalogue, err := c.getCatalogue()
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

// CheckInterfaceScale is an error naming the scale where the interface scale setting does not take
// it.
func CheckInterfaceScale(name string, scale float64) error {
	return settings.InterfaceScaleBounds.Check(name, scale)
}

// CheckTextScale is an error naming the scale where the text scale setting does not take it.
func CheckTextScale(name string, scale float64) error {
	return settings.TextScaleBounds.Check(name, scale)
}

// appearances is the settings file as the themes reach it, with what the
// command line said about size standing over what the file holds.
type appearances struct {
	cfg    Config
	scales *scales
	say    func(string)
}

func (a appearances) Read() (theme.Appearance, error) { return a.cfg.readAppearance(a.scales) }

func (a appearances) Write(choice theme.Appearance) error {
	return a.cfg.writeAppearance(choice, a.scales)
}

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
func (l *scales) apply(appearance *theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.uiScale > 0 {
		appearance.InterfaceScale = l.uiScale
	}
	if l.textScale > 0 {
		appearance.TextScale = l.textScale
	}
}

// clearSizes clears CLI scale overrides when the user configures explicit values.
func (l *scales) clearSizes(current theme.Appearance) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if current.InterfaceScale > 0 {
		l.uiScale = 0
	}
	if current.TextScale > 0 {
		l.textScale = 0
	}
}

func (c Config) getCatalogue() (theme.Catalogue, error) {
	if c.ThemesPath != "" {
		return theme.OpenAt(c.ThemesPath)
	}
	if folder, selected := c.getPathBeside("themes"); selected {
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
	aged := theme.Appearance{
		ThemeName:      activeSettings.Appearance.Theme,
		Mode:           settings.Mode(activeSettings.Appearance.Mode),
		InterfaceScale: activeSettings.Appearance.InterfaceScale,
		TextScale:      activeSettings.Appearance.TextScale,
	}
	scales.apply(&aged)
	return aged, nil
}

const appearancePrefix = "appearance"

// writeAppearance writes a choice into the file. Both sizes are checked before any of it
// is written, so a number outside what its setting goes to leaves the file as
// it stands.
func (c Config) writeAppearance(selectedVal theme.Appearance, scales *scales) error {
	path, err := c.settingsFile()
	if err != nil {
		return err
	}
	writing := []settings.Setting{
		{At: []string{appearancePrefix, "theme"}, Written: selectedVal.ThemeName},
		{At: []string{appearancePrefix, "mode"}, Written: settings.GetModeName(selectedVal.Mode)},
	}
	if selectedVal.InterfaceScale > 0 {
		err := settings.InterfaceScaleBounds.Check("appearance.interface_scale", selectedVal.InterfaceScale)
		if err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{appearancePrefix, "interface_scale"}, Written: selectedVal.InterfaceScale})
	}
	if selectedVal.TextScale > 0 {
		if err := settings.TextScaleBounds.Check("appearance.text_scale", selectedVal.TextScale); err != nil {
			return err
		}
		writing = append(writing,
			settings.Setting{At: []string{appearancePrefix, "text_scale"}, Written: selectedVal.TextScale})
	}
	into := DefaultSettings()
	if err := settings.Save(path, &into, writing...); err != nil {
		return err
	}
	scales.clearSizes(selectedVal)
	return nil
}
