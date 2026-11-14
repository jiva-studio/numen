package settings_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

// The appearance is the theme, the mode word read as a scheme and the two scales of the file.
func TestReadAppearanceReadsTheAppearanceOfTheFile(t *testing.T) {
	file := &memoryFile{current: stored{values: map[string]any{
		"app.appearance.theme":          "preset:night",
		"app.appearance.mode":           "dark",
		"app.appearance.interfaceScale": 1.25,
		"app.appearance.textScale":      1.0,
	}}}

	got, err := settings.NewReadAppearance(file).Execute()

	want := themedomain.Appearance{ThemeName: "preset:night", Mode: themedomain.Dark, InterfaceScale: 1.25, TextScale: 1}
	if err != nil || got != want {
		t.Fatalf("appearance = %+v, %v, want %+v", got, err, want)
	}
}

// A scale of zero is not writes, so the stored scale stays.
func TestWriteAppearanceWritesTheModeAsAWordAndSkipsAScaleOfZero(t *testing.T) {
	file := &memoryFile{}
	declared := [][]string{{"app", "appearance", "theme"}, {"app", "appearance", "mode"}, {"app", "appearance", "textScale"}}
	write := settings.NewWriteSettings(settings.NewCheckSettings(jsonCodec{}, refusingValidator{declared: declared}, none.PluginSettings{}), file, refreshRecorder{new([]string)})

	err := settings.NewWriteAppearance(jsonCodec{}, write).Execute(themedomain.Appearance{ThemeName: "preset:night", Mode: themedomain.Light, TextScale: 1.5})

	var got []string
	for _, one := range file.writes {
		got = append(got, one.Path[2]+"="+one.JSON)
	}
	if want := []string{"theme=\"preset:night\"", "mode=\"light\"", "textScale=1.5"}; err != nil || !slices.Equal(got, want) {
		t.Fatalf("writes = %v, %v, want %v", got, err, want)
	}
}
