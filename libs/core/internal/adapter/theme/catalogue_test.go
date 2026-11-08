package theme_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/watchhub"
)

func createThemeFolder(t *testing.T) theme.Catalog {
	t.Helper()
	catalog, err := theme.OpenAt(filepath.Join(t.TempDir(), "numen", "themes"), watchhub.NewHub())
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func writeTheme(t *testing.T, catalog theme.Catalog, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(catalog.Dir(), name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func getNames(themes []theme.Theme) []string {
	names := make([]string, 0, len(themes))
	for _, one := range themes {
		names = append(names, one.Name)
	}
	return names
}

func hasTheme(themes []theme.Theme, name string) bool {
	for _, one := range themes {
		if one.Name == name {
			return true
		}
	}
	return false
}

// The theme an unconfigured installation is set to, and the theme a name matching
// nothing falls back to, are one palette.
func TestTheSettingAnInstallationStartsOnIsThisProductsOwnPalette(t *testing.T) {
	if settings.DefaultTheme != theme.Default {
		t.Errorf("the settings start on %q and a name matching nothing falls back to %q",
			settings.DefaultTheme, theme.Default)
	}
}

func TestTheFolderIsMadeEmptyOnTheFirstRun(t *testing.T) {
	catalog := createThemeFolder(t)
	entries, err := os.ReadDir(catalog.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the folder was made holding %d files", len(entries))
	}
	for _, one := range catalog.Themes() {
		if one.Origin == theme.Mine {
			t.Errorf("an empty folder offers %q", one.Name)
		}
	}
}

func TestOnlyThisProductsOwnPaletteShipsInsideTheApplication(t *testing.T) {
	catalog := createThemeFolder(t)
	var shipped []string
	for _, one := range catalog.Themes() {
		if one.Origin == theme.Preset {
			shipped = append(shipped, one.Name)
		}
	}
	if len(shipped) != 1 || shipped[0] != theme.Default {
		t.Errorf("the binary ships %v", shipped)
	}
	for _, name := range []string{"preset:dracula", "preset:nord", "preset:solarized"} {
		if _, err := catalog.Text(name); !errors.Is(err, theme.ErrNoTheme) {
			t.Errorf("%s: error = %v, want ErrNoTheme", name, err)
		}
	}
}

// The mode selects this product's own palette, which declares nothing about
// `color-scheme`.
func TestThisProductsOwnPalettePreservesTheModeSomethingToChoose(t *testing.T) {
	catalog := createThemeFolder(t)
	text, err := catalog.Text(theme.Default)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "color-scheme") {
		t.Error("this product's own palette pins the mode")
	}
	for _, one := range catalog.Themes() {
		if one.Name == theme.Default && one.IsPinned {
			t.Error("this product's own palette is offered as pinned")
		}
	}
}

func TestAPersonsThemeExistsAdjacentAPluginThemeOfTheSameFilename(t *testing.T) {
	catalog := createThemeFolder(t).WithPlugins(createFakePlugins())
	writeTheme(t, catalog, "night.css", ":root { --numen-surface: #000000 }")

	themes := catalog.Themes()
	pluginName := theme.PluginName(pluginID, "night")
	if !hasTheme(themes, pluginName) || !hasTheme(themes, "mine:night") {
		t.Fatalf("one hides the other: %v", getNames(themes))
	}
	text, err := catalog.Text("mine:night")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "#000000") {
		t.Errorf("the person's file reads %q", text)
	}
}

// One level, and names ending `.css`.
func TestOnlyACssFileInTheFolderIsATheme(t *testing.T) {
	catalog := createThemeFolder(t)
	writeTheme(t, catalog, "dracula.css.bak", ":root {}")
	writeTheme(t, catalog, "README.md", "the themes I wrote")
	if err := os.MkdirAll(filepath.Join(catalog.Dir(), "kept.css", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTheme(t, catalog, filepath.Join("kept.css", "nord.css"), ":root {}")

	for _, one := range catalog.Themes() {
		if one.Origin == theme.Mine {
			t.Errorf("the folder offers %q", one.Name)
		}
	}
	if _, err := catalog.Text("mine:nord"); err == nil {
		t.Error("a file one level down was read")
	}
}

func TestANameThatWouldLeaveTheFolderIsRefused(t *testing.T) {
	catalog := createThemeFolder(t)
	secret := filepath.Join(filepath.Dir(catalog.Dir()), "numen.json")
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
		if text, err := catalog.Text(name); err == nil {
			t.Errorf("%s was read as %q", name, text)
		}
	}
}

func TestAFileTooLargeToSpliceIntoThePageIsNoTheme(t *testing.T) {
	catalog := createThemeFolder(t)
	writeTheme(t, catalog, "vast.css", strings.Repeat("a", theme.MaxSize+1))
	writeTheme(t, catalog, "small.css", strings.Repeat("a", theme.MaxSize))

	themes := catalog.Themes()
	if hasTheme(themes, "mine:vast") {
		t.Error("a file past the bound is offered")
	}
	if !hasTheme(themes, "mine:small") {
		t.Errorf("a file at the bound is not offered: %v", getNames(themes))
	}
	if _, err := catalog.Text("mine:vast"); err == nil {
		t.Error("a file past the bound was read")
	}
}

func TestGetApplied_UnknownNameFallsBackToDefaultAndReportsName(t *testing.T) {
	catalog := createThemeFolder(t)
	applied, missing := catalog.GetApplied("mine:the-one-i-deleted")
	if applied != theme.Default {
		t.Errorf("applied %q", applied)
	}
	if missing != "mine:the-one-i-deleted" {
		t.Errorf("says %q could not be found", missing)
	}

	writeTheme(t, catalog, "kept.css", ":root {}")
	if applied, missing := catalog.GetApplied("mine:kept"); applied != "mine:kept" || missing != "" {
		t.Errorf("applied %q, missing %q", applied, missing)
	}
	if applied, missing := catalog.GetApplied(""); applied != theme.Default || missing != "" {
		t.Errorf("an installation with no themes applied %q, missing %q", applied, missing)
	}
}

// Nothing the disk answers here stops the window: what is left is the presets.
func TestAFolderThatCannotBeReadIsAListOfWhatShips(t *testing.T) {
	catalog := createThemeFolder(t)
	if os.Geteuid() == 0 {
		t.Skip("root reads a folder whatever its mode says")
	}
	if err := os.Chmod(catalog.Dir(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(catalog.Dir(), 0o755); err != nil {
			t.Error(err)
		}
	})

	themes := catalog.Themes()
	if !hasTheme(themes, theme.Default) {
		t.Errorf("no themes: %v", getNames(themes))
	}
	for _, one := range themes {
		if one.Origin == theme.Mine {
			t.Errorf("a folder that cannot be read offered %q", one.Name)
		}
	}
	if applied, _ := catalog.GetApplied("mine:dracula"); applied != theme.Default {
		t.Errorf("applied %q", applied)
	}
}

// A machine with no configuration folder applies the theme that ships.
func TestCatalogue_NoFolderYieldsBundledThemes(t *testing.T) {
	var catalog theme.Catalog
	themes := catalog.Themes()
	if !hasTheme(themes, theme.Default) {
		t.Errorf("no themes: %v", getNames(themes))
	}
	if _, err := catalog.Text("mine:dracula"); err == nil {
		t.Error("a theme was read out of a folder that is not there")
	}
	if _, err := catalog.Text(theme.Default); err != nil {
		t.Errorf("this product's own palette: %v", err)
	}
}

// A `.css` that is a link to nothing is a file in the folder and not a theme.
func TestCatalogue_UnreadableThemeIsExcluded(t *testing.T) {
	catalog := createThemeFolder(t)
	if err := os.Symlink(filepath.Join(catalog.Dir(), "gone.css"),
		filepath.Join(catalog.Dir(), "dangling.css")); err != nil {
		t.Skipf("no links on this filesystem: %v", err)
	}
	for _, one := range catalog.Themes() {
		if one.Name == "mine:dangling" {
			t.Error("a link to nothing is offered as a theme")
		}
	}
	if _, err := catalog.Text("mine:dangling"); err == nil {
		t.Error("a link to nothing was read")
	}
}
