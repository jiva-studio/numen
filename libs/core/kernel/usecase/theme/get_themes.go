package theme

import (
	"fmt"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/themeport"
)

// AsDesigned is the multiplier that draws everything the size it was drawn at.
const AsDesigned = 1

// Bounds is the allowed range of a scale setting, reported to clients that prompt for a number.
type Bounds struct{ Min, Max float64 }

// Themes is every theme, the one applied and the appearance it is applied with.
type Themes struct {
	List       []themedomain.Theme
	Applied    string
	Appearance themedomain.Appearance

	InterfaceScaleBounds, TextScaleBounds Bounds
}

// GetThemes lists the themes with the stored appearance and the theme it applies.
type GetThemes struct {
	catalog                               catalog
	reader                                themeport.AppearanceReader
	warnings                              themeport.ThemeWarnings
	scales                                *Scales
	interfaceScaleBounds, textScaleBounds Bounds
}

// NewGetThemes makes the listing over the sources, the stored appearance, and the bounds of the two scales.
func NewGetThemes(
	sources Sources,
	reader themeport.AppearanceReader,
	warnings themeport.ThemeWarnings,
	scales *Scales,
	interfaceScaleBounds, textScaleBounds Bounds,
) *GetThemes {
	return &GetThemes{
		catalog:              catalog{sources: sources},
		reader:               reader,
		warnings:             warnings,
		scales:               scales,
		interfaceScaleBounds: interfaceScaleBounds,
		textScaleBounds:      textScaleBounds,
	}
}

// Execute returns every theme and the one applied. A stored theme that no longer resolves applies the
// default and is warned about; the warning is withdrawn once the stored theme is found.
func (u *GetThemes) Execute() Themes {
	current := u.getAppearance()
	applied, missing := u.catalog.getAppliedTheme(current.ThemeName)
	if missing != "" {
		u.warnings.WarnTheme(fmt.Sprintf("Theme not found: %s. Applied: %s", missing, applied))
	} else {
		u.warnings.WarnTheme("")
	}
	return Themes{
		List:                 u.catalog.list(),
		Applied:              applied,
		Appearance:           current,
		InterfaceScaleBounds: u.interfaceScaleBounds,
		TextScaleBounds:      u.textScaleBounds,
	}
}

func (u *GetThemes) getAppearance() themedomain.Appearance {
	current := themedomain.Appearance{
		ThemeName:      themedomain.Default,
		Mode:           themedomain.System,
		InterfaceScale: AsDesigned,
		TextScale:      AsDesigned,
	}
	stored, err := u.reader.ReadAppearance()
	if err != nil {
		u.warnings.WarnTheme(fmt.Sprintf("Settings not read, applied %s: %v", themedomain.Default, err))
		return current
	}
	u.scales.apply(&stored)
	if stored.ThemeName != "" {
		current.ThemeName = stored.ThemeName
	}
	current.Mode = stored.Mode
	if stored.InterfaceScale > 0 {
		current.InterfaceScale = stored.InterfaceScale
	}
	if stored.TextScale > 0 {
		current.TextScale = stored.TextScale
	}
	return current
}
