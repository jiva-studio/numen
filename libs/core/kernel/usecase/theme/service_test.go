package theme_test

import (
	"errors"
	"strings"
	"testing"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
)

var errNotWritten = errors.New("the settings are read-only")

func TestTheListReturnsWhatIsAppliedAndWhichHalfItIsReadAs(t *testing.T) {
	night := themedomain.PluginName(pluginID, "night")
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: night, Mode: themedomain.Dark})
	desk.writeTheme(t, "mine.css", ":root { --numen-surface: #000000 }")

	answer := desk.getThemes()
	if answer.Applied != night || answer.Appearance.Mode != themedomain.Dark {
		t.Errorf("applied %q, read as %v", answer.Applied, answer.Appearance.Mode)
	}

	origins := map[string]themedomain.Origin{}
	for _, one := range answer.List {
		origins[one.Name] = one.Origin
	}
	if origins[themedomain.Default] != themedomain.Preset {
		t.Errorf("%s came off %v", themedomain.Default, origins[themedomain.Default])
	}
	if origins[night] != themedomain.Plugin {
		t.Errorf("%s came off %v", night, origins[night])
	}
	if origins["mine:mine"] != themedomain.Mine {
		t.Errorf("the person's theme came off %v", origins["mine:mine"])
	}
}

func TestReadThemeResolvesKnownNamesAndEmptyForOthers(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})
	desk.writeTheme(t, "kept.css", ":root { --numen-surface: #123456 }")

	for name, want := range map[string]string{
		"mine:kept": "#123456",
		themedomain.PluginName(pluginID, "night"): "--numen-surface",
		themedomain.Default:                       "own palette",
		"preset:dracula":                          "",
		"mine:../../../.ssh/id_rsa":               "",
		"mine:gone":                               "",
	} {
		text, err := desk.readTheme(name)
		if err != nil {
			t.Fatal(err)
		}
		if want == "" && text != "" {
			t.Errorf("%s answered with %q", name, text)
		}
		if want != "" && !strings.Contains(text, want) {
			t.Errorf("%s answered with %q", name, text)
		}
	}
}

func TestReadThemeOfABuildWithoutAFolderStillReadsThePresets(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.sources.Own = themefiles.Files{}

	if text, err := desk.readTheme(themedomain.Default); err != nil || text == "" {
		t.Errorf("text = %q, error = %v", text, err)
	}
	if text, err := desk.readTheme("mine:dracula"); err != nil || text != "" {
		t.Errorf("text = %q, error = %v, want nothing", text, err)
	}
}

func TestChoosingWritesTheThemeAndTheHalfItIsReadAs(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})
	desk.writeTheme(t, "nord.css", ":root {}")
	choice := themedomain.Appearance{ThemeName: "mine:nord", Mode: themedomain.Dark}
	if err := desk.writeAppearance(choice); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(desk.appearance.saves) != 1 || desk.appearance.saves[0] != choice {
		t.Fatalf("written: %+v", desk.appearance.saves)
	}
	if answer := desk.getThemes(); answer.Applied != "mine:nord" {
		t.Errorf("applied %q", answer.Applied)
	}
}

// A client reports the range of each size before the user types a number into it.
func TestTheListReturnsTheTwoSizesAndHowFarEachGoes(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default, InterfaceScale: 1.25, TextScale: 1.5})

	answer := desk.getThemes()
	if answer.Appearance.InterfaceScale != 1.25 || answer.Appearance.TextScale != 1.5 {
		t.Errorf("drawn at %v and set at %v", answer.Appearance.InterfaceScale, answer.Appearance.TextScale)
	}
	if answer.InterfaceScaleBounds != (themeusecase.Bounds{Min: 0.8, Max: 2}) {
		t.Errorf("the interface goes %v", answer.InterfaceScaleBounds)
	}
	if answer.TextScaleBounds != (themeusecase.Bounds{Min: 0.8, Max: 1.75}) {
		t.Errorf("the text goes %v", answer.TextScaleBounds)
	}
}

// A window with no theme applied is drawn at the size it was designed at.
func TestASettingsFileNamingNoSizeIsGeneratedAsDesigned(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})

	answer := desk.getThemes()
	if answer.Appearance.InterfaceScale != themeusecase.AsDesigned || answer.Appearance.TextScale != themeusecase.AsDesigned {
		t.Errorf("drawn at %v and set at %v", answer.Appearance.InterfaceScale, answer.Appearance.TextScale)
	}
}

// A scale given on the command line applies until the user chooses one explicitly.
func TestACommandLineScaleIsDroppedByAnExplicitScale(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default, InterfaceScale: 1.25, TextScale: 1.5})
	desk.scales = themeusecase.NewScales(1.75, 0)

	if got := desk.getThemes().Appearance; got.InterfaceScale != 1.75 || got.TextScale != 1.5 {
		t.Fatalf("drawn at %v and set at %v", got.InterfaceScale, got.TextScale)
	}
	if err := desk.writeAppearance(themedomain.Appearance{ThemeName: themedomain.Default, TextScale: 1.1}); err != nil {
		t.Fatal(err)
	}
	if got := desk.getThemes().Appearance; got.InterfaceScale != 1.75 {
		t.Errorf("choosing the text size dropped the interface size on the command line: %v", got.InterfaceScale)
	}
	if err := desk.writeAppearance(themedomain.Appearance{ThemeName: themedomain.Default, InterfaceScale: 1.3}); err != nil {
		t.Fatal(err)
	}
	if got := desk.getThemes().Appearance; got.InterfaceScale != 1.3 {
		t.Errorf("drawn at %v after the user chose 1.3", got.InterfaceScale)
	}
}

// One size is one command, and choosing it leaves the other unchanged.
func TestChoosingOneSizePreservesTheOtherAsItExists(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default, InterfaceScale: 1.25, TextScale: 1.5})
	desk.writeTheme(t, "nord.css", ":root {}")

	err := desk.writeAppearance(themedomain.Appearance{ThemeName: "mine:nord", Mode: themedomain.Dark, InterfaceScale: 1.75})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if len(desk.appearance.saves) != 1 || desk.appearance.saves[0].InterfaceScale != 1.75 || desk.appearance.saves[0].TextScale != 0 {
		t.Fatalf("written: %+v", desk.appearance.saves)
	}
}

// The settings are unchanged, and the user is left with the list that was there.
func TestAChoiceThatCannotBeAppliedIsRefusedAndReturnsWhy(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})
	for _, name := range []string{"mine:the-one-i-deleted", "mine:../../../.ssh/id_rsa", "preset:nord"} {
		err := desk.writeAppearance(themedomain.Appearance{ThemeName: name, Mode: themedomain.Dark})
		if err == nil {
			t.Errorf("%s was written", name)
			continue
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%s failed with %q", name, err)
		}
	}
	if len(desk.appearance.saves) != 0 {
		t.Errorf("written: %+v", desk.appearance.saves)
	}
}

func TestASettingsFileThatWouldNotTakeTheChoiceReturnsSo(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: themedomain.Default})
	desk.appearance.failure = errNotWritten
	if err := desk.writeAppearance(themedomain.Appearance{ThemeName: themedomain.Default, Mode: themedomain.Dark}); !errors.Is(err, errNotWritten) {
		t.Errorf("failed with %v", err)
	}
}

// A theme deleted out of the folder does not leave the window undressed.
func TestMissingSettingsThemeReportsAndUsesDefault(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: "mine:the-one-i-deleted", Mode: themedomain.Light})
	answer := desk.getThemes()
	if answer.Applied != themedomain.Default {
		t.Errorf("applied %q", answer.Applied)
	}
	if answer.Appearance.Mode != themedomain.Light {
		t.Errorf("read as %v", answer.Appearance.Mode)
	}
	if len(desk.appearance.warnings) != 1 || !strings.Contains(desk.appearance.warnings[0], "mine:the-one-i-deleted") {
		t.Errorf("said %v", desk.appearance.warnings)
	}
}

// The warning about a missing theme is withdrawn once the theme the settings name is found again.
func TestFoundThemeWithdrawsTheEarlierWarning(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: "mine:the-one-i-deleted", Mode: themedomain.Light})
	desk.getThemes()
	desk.appearance.appearance = themedomain.Appearance{ThemeName: themedomain.Default, Mode: themedomain.Light}
	desk.getThemes()
	if len(desk.appearance.warnings) != 2 || desk.appearance.warnings[1] != "" {
		t.Errorf("said %v, want the warning and then an empty text", desk.appearance.warnings)
	}
}

// Choosing a theme that is found withdraws the warning about the one that was missing.
func TestWritingAFoundThemeWithdrawsTheEarlierWarning(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: "mine:the-one-i-deleted", Mode: themedomain.Light})
	desk.getThemes()
	if err := desk.writeAppearance(themedomain.Appearance{ThemeName: themedomain.Default, Mode: themedomain.Light}); err != nil {
		t.Fatal(err)
	}
	if len(desk.appearance.warnings) != 2 || desk.appearance.warnings[1] != "" {
		t.Errorf("said %v, want the warning and then an empty text", desk.appearance.warnings)
	}
}
