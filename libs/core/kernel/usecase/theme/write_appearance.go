package theme

import (
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/themeport"
)

// WriteAppearance stores the theme, the color scheme and the scales the user chose.
type WriteAppearance struct {
	catalog  catalog
	writer   themeport.AppearanceWriter
	warnings themeport.ThemeWarnings
	scales   *Scales
}

// NewWriteAppearance makes the write over the sources and the store of the appearance.
func NewWriteAppearance(
	sources Sources,
	writer themeport.AppearanceWriter,
	warnings themeport.ThemeWarnings,
	scales *Scales,
) *WriteAppearance {
	return &WriteAppearance{catalog: catalog{sources: sources}, writer: writer, warnings: warnings, scales: scales}
}

// Execute stores the appearance when its theme can be read. It returns the reason, and writes
// nothing, where no theme answers for the name or the store refuses the values.
func (u *WriteAppearance) Execute(chosen themedomain.Appearance) error {
	if _, err := u.catalog.read(chosen.ThemeName); err != nil {
		return err
	}
	if err := u.writer.WriteAppearance(chosen); err != nil {
		return err
	}
	u.scales.clear(chosen)
	// The theme chosen is found, so a warning about the one that was missing is withdrawn.
	u.warnings.WarnTheme("")
	return nil
}
