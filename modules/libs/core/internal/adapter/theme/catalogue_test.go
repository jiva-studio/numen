package theme_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/theme"
)

// folder is a themes folder of a person's, made the way the first run makes it.
func folder(t *testing.T) theme.Catalogue {
	t.Helper()
	catalogue, err := theme.OpenAt(filepath.Join(t.TempDir(), "numen", "themes"))
	if err != nil {
		t.Fatal(err)
	}
	return catalogue
}

func put(t *testing.T, catalogue theme.Catalogue, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(catalogue.Dir(), name), []byte(body), 0o644); err != nil {
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

func holds(themes []theme.Theme, name string) bool {
	for _, one := range themes {
		if one.Name == name {
			return true
		}
	}
	return false
}

// The theme an installation nobody has dressed is set to, and the theme a name
// matching nothing falls back to, are one palette.
func TestTheSettingAnInstallationStartsOnIsThisProductsOwnPalette(t *testing.T) {
	if settings.DefaultTheme != theme.Default {
		t.Errorf("the settings start on %q and a name matching nothing wears %q",
			settings.DefaultTheme, theme.Default)
	}
}

func TestTheFolderIsMadeEmptyOnTheFirstRun(t *testing.T) {
	catalogue := folder(t)
	entries, err := os.ReadDir(catalogue.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the folder was made holding %d files", len(entries))
	}
	for _, one := range catalogue.Themes() {
		if one.Shelf == theme.Mine {
			t.Errorf("an empty folder offers %q", one.Name)
		}
	}
}

func TestThirteenPalettesShipInsideTheApplication(t *testing.T) {
	themes := folder(t).Themes()
	for _, name := range []string{
		"preset:numen", "preset:dracula", "preset:nord", "preset:solarized",
		"preset:catppuccin", "preset:gruvbox", "preset:amber", "preset:github",
		"preset:one-dark", "preset:monokai", "preset:tokyo-night", "preset:ayu",
		"preset:cobalt2",
	} {
		if !holds(themes, name) {
			t.Errorf("no %s among %v", name, getNames(themes))
		}
	}
}

// The mode chooses for this product's own palette, so that palette says nothing
// about `color-scheme`.
func TestThisProductsOwnPaletteLeavesTheModeSomethingToChoose(t *testing.T) {
	catalogue := folder(t)
	text, err := catalogue.Text(theme.Default)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(text, "color-scheme") {
		t.Error("this product's own palette pins the mode")
	}
	for _, one := range catalogue.Themes() {
		if one.Name == theme.Default && one.Pinned {
			t.Error("this product's own palette is offered as pinned")
		}
	}
}

// A palette published in one half pins it; one published in two halves does not,
// and the setting chooses.
func TestAThemeSaysWhetherItIsAPair(t *testing.T) {
	pinned := map[string]bool{"preset:dracula": true, "preset:solarized": false}
	for _, one := range folder(t).Themes() {
		want, asked := pinned[one.Name]
		if asked && one.Pinned != want {
			t.Errorf("%s is pinned: %v", one.Name, one.Pinned)
		}
	}
}

func TestAPersonsThemeStandsBesideThePresetItsNameIsShared(t *testing.T) {
	catalogue := folder(t)
	put(t, catalogue, "dracula.css", ":root { --numen-surface: #000000 }")

	themes := catalogue.Themes()
	if !holds(themes, "preset:dracula") || !holds(themes, "mine:dracula") {
		t.Fatalf("one hides the other: %v", getNames(themes))
	}
	text, err := catalogue.Text("mine:dracula")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "#000000") {
		t.Errorf("the person's file reads %q", text)
	}
}

// One level, and names ending `.css`.
func TestOnlyACssFileInTheFolderIsATheme(t *testing.T) {
	catalogue := folder(t)
	put(t, catalogue, "dracula.css.bak", ":root {}")
	put(t, catalogue, "README.md", "the themes I wrote")
	if err := os.MkdirAll(filepath.Join(catalogue.Dir(), "kept.css", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	put(t, catalogue, filepath.Join("kept.css", "nord.css"), ":root {}")

	for _, one := range catalogue.Themes() {
		if one.Shelf == theme.Mine {
			t.Errorf("the folder offers %q", one.Name)
		}
	}
	if _, err := catalogue.Text("mine:nord"); err == nil {
		t.Error("a file one level down was read")
	}
}

func TestANameThatWouldLeaveTheFolderIsRefused(t *testing.T) {
	catalogue := folder(t)
	secret := filepath.Join(filepath.Dir(catalogue.Dir()), "numen.json")
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
		"shelfless:dracula",
	} {
		if text, err := catalogue.Text(name); err == nil {
			t.Errorf("%s was read as %q", name, text)
		}
	}
}

func TestAFileTooLargeToSpliceIntoThePageIsNoTheme(t *testing.T) {
	catalogue := folder(t)
	put(t, catalogue, "vast.css", strings.Repeat("a", theme.MaxSize+1))
	put(t, catalogue, "small.css", strings.Repeat("a", theme.MaxSize))

	themes := catalogue.Themes()
	if holds(themes, "mine:vast") {
		t.Error("a file past the bound is offered")
	}
	if !holds(themes, "mine:small") {
		t.Errorf("a file at the bound is not offered: %v", getNames(themes))
	}
	if _, err := catalogue.Text("mine:vast"); err == nil {
		t.Error("a file past the bound was read")
	}
}

func TestANameMatchingNothingWearsThisProductsPaletteAndSaysWhichNameItWas(t *testing.T) {
	catalogue := folder(t)
	applied, missing := catalogue.GetApplied("mine:the-one-i-deleted")
	if applied != theme.Default {
		t.Errorf("wears %q", applied)
	}
	if missing != "mine:the-one-i-deleted" {
		t.Errorf("says %q could not be found", missing)
	}

	put(t, catalogue, "kept.css", ":root {}")
	if applied, missing := catalogue.GetApplied("mine:kept"); applied != "mine:kept" || missing != "" {
		t.Errorf("wears %q, missing %q", applied, missing)
	}
	if applied, missing := catalogue.GetApplied(""); applied != theme.Default || missing != "" {
		t.Errorf("an installation nobody dressed wears %q, missing %q", applied, missing)
	}
}

// Nothing the disk answers here stops the window: what is left is the presets.
func TestAFolderThatCannotBeReadIsAListOfWhatShips(t *testing.T) {
	catalogue := folder(t)
	if os.Geteuid() == 0 {
		t.Skip("root reads a folder whatever its mode says")
	}
	if err := os.Chmod(catalogue.Dir(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(catalogue.Dir(), 0o755) })

	themes := catalogue.Themes()
	if !holds(themes, theme.Default) {
		t.Errorf("nothing to wear: %v", getNames(themes))
	}
	for _, one := range themes {
		if one.Shelf == theme.Mine {
			t.Errorf("a folder that cannot be read offered %q", one.Name)
		}
	}
	if applied, _ := catalogue.GetApplied("mine:dracula"); applied != theme.Default {
		t.Errorf("wears %q", applied)
	}
}

// A machine with no configuration folder is a window wearing what ships.
func TestACatalogueWithNoFolderOffersWhatShips(t *testing.T) {
	var catalogue theme.Catalogue
	themes := catalogue.Themes()
	if !holds(themes, theme.Default) {
		t.Errorf("nothing to wear: %v", getNames(themes))
	}
	if _, err := catalogue.Text("mine:dracula"); err == nil {
		t.Error("a theme was read out of a folder that is not there")
	}
	if _, err := catalogue.Text(theme.Default); err != nil {
		t.Errorf("this product's own palette: %v", err)
	}
}

// A `.css` that is a link to nothing is a file in the folder and not a theme.
func TestAThemeThatCannotBeReadIsNotOffered(t *testing.T) {
	catalogue := folder(t)
	if err := os.Symlink(filepath.Join(catalogue.Dir(), "gone.css"),
		filepath.Join(catalogue.Dir(), "dangling.css")); err != nil {
		t.Skipf("no links on this filesystem: %v", err)
	}
	for _, one := range catalogue.Themes() {
		if one.Name == "mine:dangling" {
			t.Error("a link to nothing is offered as a theme")
		}
	}
	if _, err := catalogue.Text("mine:dangling"); err == nil {
		t.Error("a link to nothing was read")
	}
}
