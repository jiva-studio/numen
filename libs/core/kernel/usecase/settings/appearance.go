package settings

import (
	"encoding/json"
	"fmt"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

func appearancePath(key string) []string {
	return []string{settingsdomain.AppOwner, "appearance", key}
}

// ReadAppearance reads the theme, the mode and the scales the settings file holds.
type ReadAppearance struct {
	file settingsport.SettingsFile
}

// NewReadAppearance makes the use case over the settings file.
func NewReadAppearance(file settingsport.SettingsFile) *ReadAppearance {
	return &ReadAppearance{file: file}
}

// Execute returns the appearance as the file holds it.
func (u *ReadAppearance) Execute() (themedomain.Appearance, error) {
	stored, err := u.file.ReadUnchecked()
	if err != nil {
		return themedomain.Appearance{}, err
	}
	themeName, _ := getAppearanceValue(stored, "theme").(string)
	mode, _ := getAppearanceValue(stored, "mode").(string)
	interfaceScale, _ := getAppearanceValue(stored, "interfaceScale").(float64)
	textScale, _ := getAppearanceValue(stored, "textScale").(float64)
	return themedomain.Appearance{
		ThemeName:      themeName,
		Mode:           themedomain.ParseColorScheme(mode),
		InterfaceScale: interfaceScale,
		TextScale:      textScale,
	}, nil
}

func getAppearanceValue(stored settingsdomain.Stored, key string) any {
	value, _ := stored.GetValue(appearancePath(key)...)
	return value
}

// WriteAppearance writes the theme, the mode and the scales into the settings file. A scale of zero
// leaves the stored one as it is.
type WriteAppearance struct {
	write *WriteSettings
}

// NewWriteAppearance makes the use case over the settings writer.
func NewWriteAppearance(write *WriteSettings) *WriteAppearance {
	return &WriteAppearance{write: write}
}

// Execute writes the appearance.
func (u *WriteAppearance) Execute(selected themedomain.Appearance) error {
	values := []struct {
		path  []string
		value any
	}{
		{appearancePath("theme"), selected.ThemeName},
		{appearancePath("mode"), selected.Mode.GetName()},
	}
	if selected.InterfaceScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearancePath("interfaceScale"), selected.InterfaceScale})
	}
	if selected.TextScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{appearancePath("textScale"), selected.TextScale})
	}
	writing := make([]settingsdomain.Setting, 0, len(values))
	for _, one := range values {
		encoded, err := json.Marshal(one.value)
		if err != nil {
			return fmt.Errorf("%s: %w", strings.Join(one.path, "."), err)
		}
		writing = append(writing, settingsdomain.Setting{Path: one.path, JSON: string(encoded)})
	}
	return u.write.Execute(writing)
}
