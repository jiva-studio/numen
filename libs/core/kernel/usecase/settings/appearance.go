package settings

import (
	"fmt"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

func getAppearancePath(key string) []string {
	return []string{settingsdomain.AppOwner, "appearance", key}
}

// ReadAppearance reads the theme, the mode and the scales the settings file holds.
type ReadAppearance struct {
	file SettingsFile
}

// NewReadAppearance makes the use case over the settings file.
func NewReadAppearance(file SettingsFile) *ReadAppearance {
	return &ReadAppearance{file: file}
}

// Execute returns the appearance as the file holds it.
func (u *ReadAppearance) Execute() (themedomain.Appearance, error) {
	stored, err := u.file.ReadUnchecked()
	if err != nil {
		return themedomain.Appearance{}, err
	}
	themeName := getAppearanceValue[string](stored, "theme")
	mode := getAppearanceValue[string](stored, "mode")
	interfaceScale := getAppearanceValue[float64](stored, "interfaceScale")
	textScale := getAppearanceValue[float64](stored, "textScale")
	return themedomain.Appearance{
		ThemeName:      themeName,
		Mode:           themedomain.ParseColorScheme(mode),
		InterfaceScale: interfaceScale,
		TextScale:      textScale,
	}, nil
}

func getAppearanceValue[T any](stored settingsdomain.StoredSettings, key string) T {
	value, _ := stored.GetValue(getAppearancePath(key)...)
	typed, isTyped := value.(T)
	if !isTyped {
		var none T
		return none
	}
	return typed
}

// WriteAppearance writes the theme, the mode and the scales into the settings file. A scale of zero
// leaves the stored one as it is.
type WriteAppearance struct {
	codec ValueCodec
	write *WriteSettings
}

// NewWriteAppearance makes the use case over the settings writer.
func NewWriteAppearance(codec ValueCodec, write *WriteSettings) *WriteAppearance {
	return &WriteAppearance{codec: codec, write: write}
}

func (u *WriteAppearance) Execute(selected themedomain.Appearance) error {
	values := []struct {
		path  []string
		value any
	}{
		{getAppearancePath("theme"), selected.ThemeName},
		{getAppearancePath("mode"), selected.Mode.GetName()},
	}
	if selected.InterfaceScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{getAppearancePath("interfaceScale"), selected.InterfaceScale})
	}
	if selected.TextScale > 0 {
		values = append(values, struct {
			path  []string
			value any
		}{getAppearancePath("textScale"), selected.TextScale})
	}
	writing := make([]settingsdomain.Setting, 0, len(values))
	for _, one := range values {
		encoded, err := u.codec.Encode(one.value)
		if err != nil {
			return fmt.Errorf("%s: %w", strings.Join(one.path, "."), err)
		}
		writing = append(writing, settingsdomain.Setting{Path: one.path, JSON: encoded})
	}
	return u.write.Execute(writing)
}
