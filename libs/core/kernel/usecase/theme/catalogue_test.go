package theme_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// The theme an unconfigured installation is set to, and the theme a name matching
// nothing falls back to, are one palette.
func TestTheSettingAnInstallationStartsOnIsThisProductsOwnPalette(t *testing.T) {
	if settings.DefaultTheme != themedomain.Default {
		t.Errorf("the settings start on %q and a name matching nothing falls back to %q",
			settings.DefaultTheme, themedomain.Default)
	}
}

func TestAnEmptyFolderOffersOnlyThePresets(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	for _, one := range desk.getThemes().List {
		if one.Origin == themedomain.Mine {
			t.Errorf("an empty folder offers %q", one.Name)
		}
	}
}

func TestOnlyThisProductsOwnPaletteShipsInsideTheApplication(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	var shipped []string
	for _, one := range desk.getThemes().List {
		if one.Origin == themedomain.Preset {
			shipped = append(shipped, one.Name)
		}
	}
	if len(shipped) != 1 || shipped[0] != themedomain.Default {
		t.Errorf("the binary ships %v", shipped)
	}
	for _, name := range []string{"preset:dracula", "preset:nord", "preset:solarized"} {
		if text, err := desk.readTheme(name); err == nil && text != "" {
			t.Errorf("%s was read as %q", name, text)
		}
	}
}

// The mode selects this product's own palette, which declares nothing about
// `color-scheme`.
func TestThisProductsOwnPalettePreservesTheModeSomethingToChoose(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	text, err := desk.readTheme(themedomain.Default)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "color-scheme") {
		t.Error("this product's own palette pins the mode")
	}
	for _, one := range desk.getThemes().List {
		if one.Name == themedomain.Default && one.IsPinned {
			t.Error("this product's own palette is offered as pinned")
		}
	}
}

func TestAThemeThatDeclaresAColorSchemeIsOfferedAsPinned(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.writeTheme(t, "pinned.css", ":root { color-scheme: dark; }")
	desk.writeTheme(t, "commented.css", "/* color-scheme: dark */ :root {}")
	desk.writeTheme(t, "open.css", ":root {}")

	pinned := map[string]bool{}
	for _, one := range desk.getThemes().List {
		pinned[one.Name] = one.IsPinned
	}
	if !pinned["mine:pinned"] || pinned["mine:commented"] || pinned["mine:open"] {
		t.Errorf("pinned = %v", pinned)
	}
}

func TestAPersonsThemeExistsAdjacentAPluginThemeOfTheSameFilename(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	desk.writeTheme(t, "night.css", ":root { --numen-surface: #000000 }")

	themes := desk.getThemes().List
	pluginName := themedomain.PluginName(pluginID, "night")
	if !hasTheme(themes, pluginName) || !hasTheme(themes, "mine:night") {
		t.Fatalf("one hides the other: %v", getNames(themes))
	}
	text, err := desk.readTheme("mine:night")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "#000000") {
		t.Errorf("the person's file reads %q", text)
	}
}

func TestANameThatWouldLeaveTheFolderIsRefused(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{})
	secret := filepath.Join(filepath.Dir(desk.files.Dir()), "numen.json")
	if err := os.WriteFile(secret, []byte(`{"key":"sk-the-persons-own"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"mine:../../../.ssh/id_rsa",
		"mine:../numen",
		`mine:..\..\numen`,
		"mine:.",
		"mine:..",
		"mine:",
		"preset:../../../../etc/passwd",
		"dracula",
		"originless:dracula",
	} {
		if text, err := desk.readTheme(name); err == nil && text != "" {
			t.Errorf("%s was read as %q", name, text)
		}
		if err := desk.writeAppearance(themedomain.Appearance{ThemeName: name}); err == nil {
			t.Errorf("%s was written", name)
		}
	}
}

func TestAnUnknownNameFallsBackToDefaultAndReportsName(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: "mine:the-one-i-deleted"})
	if got := desk.getThemes().Applied; got != themedomain.Default {
		t.Errorf("applied %q", got)
	}
	if len(desk.appearance.warnings) != 1 || !strings.Contains(desk.appearance.warnings[0], "mine:the-one-i-deleted") {
		t.Errorf("said %v", desk.appearance.warnings)
	}

	desk.writeTheme(t, "kept.css", ":root {}")
	desk.appearance.appearance.ThemeName = "mine:kept"
	if got := desk.getThemes().Applied; got != "mine:kept" {
		t.Errorf("applied %q", got)
	}
	desk.appearance.appearance.ThemeName = ""
	if got := desk.getThemes().Applied; got != themedomain.Default {
		t.Errorf("an installation with no themes applied %q", got)
	}
}

// Nothing the disk answers here stops the window: what is left is the presets.
func TestAFolderThatCannotBeReadIsAListOfWhatShips(t *testing.T) {
	desk := createThemesDesk(t, themedomain.Appearance{ThemeName: "mine:dracula"})
	if os.Geteuid() == 0 {
		t.Skip("root reads a folder whatever its mode says")
	}
	if err := os.Chmod(desk.files.Dir(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(desk.files.Dir(), 0o755); err != nil {
			t.Error(err)
		}
	})

	answer := desk.getThemes()
	if !hasTheme(answer.List, themedomain.Default) {
		t.Errorf("no themes: %v", getNames(answer.List))
	}
	for _, one := range answer.List {
		if one.Origin == themedomain.Mine {
			t.Errorf("a folder that cannot be read offered %q", one.Name)
		}
	}
	if answer.Applied != themedomain.Default {
		t.Errorf("applied %q", answer.Applied)
	}
}
