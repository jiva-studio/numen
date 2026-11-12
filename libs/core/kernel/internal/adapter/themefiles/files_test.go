package themefiles_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/watchhub"
)

func createThemeFolder(t *testing.T) themefiles.Files {
	t.Helper()
	files, err := themefiles.OpenAt(filepath.Join(t.TempDir(), "numen", "themes"), watchhub.NewHub())
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func writeTheme(t *testing.T, files themefiles.Files, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(files.Dir(), name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func getTitles(files []themedomain.File) []string {
	titles := make([]string, 0, len(files))
	for _, one := range files {
		titles = append(titles, one.Title)
	}
	return titles
}

func TestTheFolderIsMadeEmptyOnTheFirstRun(t *testing.T) {
	files := createThemeFolder(t)
	entries, err := os.ReadDir(files.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the folder was made holding %d files", len(entries))
	}
	if own := files.ListOwnThemes(); len(own) != 0 {
		t.Errorf("an empty folder offers %v", getTitles(own))
	}
}

func TestOnlyThisProductsOwnPaletteShipsInsideTheApplication(t *testing.T) {
	files := createThemeFolder(t)
	shipped := getTitles(files.ListPresets())
	if len(shipped) != 1 || themedomain.GetName(themedomain.Preset, shipped[0]) != themedomain.Default {
		t.Errorf("the binary ships %v", shipped)
	}
	for _, title := range []string{"dracula", "nord", "solarized"} {
		if _, err := files.ReadPreset(title); err == nil {
			t.Errorf("%s was read as a preset", title)
		}
	}
}

// The mode selects this product's own palette, which declares nothing about
// `color-scheme`.
func TestThisProductsOwnPalettePreservesTheModeSomethingToChoose(t *testing.T) {
	files := createThemeFolder(t)
	for _, one := range files.ListPresets() {
		if strings.Contains(one.Text, "color-scheme") {
			t.Errorf("%s pins the mode", one.Title)
		}
	}
}

// One level, and names ending `.css`.
func TestOnlyACssFileInTheFolderIsATheme(t *testing.T) {
	files := createThemeFolder(t)
	writeTheme(t, files, "dracula.css.bak", ":root {}")
	writeTheme(t, files, "README.md", "the themes I wrote")
	if err := os.MkdirAll(filepath.Join(files.Dir(), "kept.css", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTheme(t, files, filepath.Join("kept.css", "nord.css"), ":root {}")

	if own := files.ListOwnThemes(); len(own) != 0 {
		t.Errorf("the folder offers %v", getTitles(own))
	}
	if _, err := files.ReadOwnTheme("nord"); err == nil {
		t.Error("a file one level down was read")
	}
}

func TestAFileTooLargeToSpliceIntoThePageIsNoTheme(t *testing.T) {
	files := createThemeFolder(t)
	writeTheme(t, files, "vast.css", strings.Repeat("a", themedomain.MaxSize+1))
	writeTheme(t, files, "small.css", strings.Repeat("a", themedomain.MaxSize))

	titles := getTitles(files.ListOwnThemes())
	if len(titles) != 1 || titles[0] != "small" {
		t.Errorf("the folder offers %v, want the file at the bound alone", titles)
	}
	if _, err := files.ReadOwnTheme("vast"); err == nil {
		t.Error("a file past the bound was read")
	}
}

func TestATitleWithNoFileIsNoTheme(t *testing.T) {
	files := createThemeFolder(t)
	if _, err := files.ReadOwnTheme("gone"); !errors.Is(err, themedomain.ErrNoTheme) {
		t.Errorf("error = %v, want ErrNoTheme", err)
	}
}

// Nothing the disk answers here stops the window: what is left is the presets.
func TestAFolderThatCannotBeReadIsAListOfNothing(t *testing.T) {
	files := createThemeFolder(t)
	if os.Geteuid() == 0 {
		t.Skip("root reads a folder whatever its mode says")
	}
	if err := os.Chmod(files.Dir(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(files.Dir(), 0o755); err != nil {
			t.Error(err)
		}
	})

	if own := files.ListOwnThemes(); len(own) != 0 {
		t.Errorf("a folder that cannot be read offered %v", getTitles(own))
	}
	if len(files.ListPresets()) == 0 {
		t.Error("the presets went with the folder")
	}
}

// A machine with no configuration folder applies the theme that ships.
func TestNoFolderYieldsBundledThemes(t *testing.T) {
	var files themefiles.Files
	if len(files.ListPresets()) == 0 {
		t.Error("no presets")
	}
	if own := files.ListOwnThemes(); len(own) != 0 {
		t.Errorf("a folder that is not there offered %v", getTitles(own))
	}
	if _, err := files.ReadOwnTheme("dracula"); !errors.Is(err, themedomain.ErrNoFolder) {
		t.Errorf("error = %v, want ErrNoFolder", err)
	}
}

// A `.css` that is a link to nothing is a file in the folder and not a theme.
func TestUnreadableThemeIsExcluded(t *testing.T) {
	files := createThemeFolder(t)
	if err := os.Symlink(filepath.Join(files.Dir(), "gone.css"),
		filepath.Join(files.Dir(), "dangling.css")); err != nil {
		t.Skipf("no links on this filesystem: %v", err)
	}
	if own := files.ListOwnThemes(); len(own) != 0 {
		t.Errorf("a link to nothing is offered as %v", getTitles(own))
	}
	if _, err := files.ReadOwnTheme("dangling"); err == nil {
		t.Error("a link to nothing was read")
	}
}
