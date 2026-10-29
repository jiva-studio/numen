package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
)

func theme(name string) settings.Setting {
	return settings.Setting{At: []string{"appearance", "theme"}, Written: name}
}

// arranged is a manually edited file: custom key order, indentation, spaced
// sections, and a number that appears in two places.
const arranged = `{
    "v": 1,

    "appearance": {
        "zoom": 1.5,
        "theme": "preset:numen",
        "mode": "system"
    },

    "indexing": {
        "embedding": {
            "indexing": {
                "use": "service",
                "service": {
                    "key": "sk-the-persons-own",
                    "name": "text-embedding-3-small"
                }
            }
        },
        "proofreading": {"max_edit_distance": 0.30}
    },

    "agent": {"use": "claude"}
}
`

// Guards against rewriting the file from a map: the output is the original file
// with the bytes of one value replaced and nothing else changed.
func TestAHandArrangedFileReachesBackWithOneValueChanged(t *testing.T) {
	path := write(t, arranged)
	if err := settings.Save(path, newDocument(), theme("mine:dracula")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Replace(arranged, `"theme": "preset:numen"`, `"theme": "mine:dracula"`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// Two settings are saved at once, and the file layout is preserved.
func TestTheThemeAndTheModeAreSavedWithoutMovingAnythingElse(t *testing.T) {
	path := write(t, arranged)
	if err := settings.Save(path, newDocument(), theme("preset:nord"),
		settings.Setting{At: []string{"appearance", "mode"}, Written: settings.ModeDark}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Replace(arranged, `"theme": "preset:numen"`, `"theme": "preset:nord"`, 1)
	want = strings.Replace(want, `"mode": "system"`, `"mode": "dark"`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// The two sizes are written in place, and a number that appears in two other
// places keeps both occurrences.
func TestTheTwoSizesAreSavedWithoutMovingAnythingElse(t *testing.T) {
	const sized = `{
    "appearance": {
        "interface_scale": 1.00,
        "text_scale": 1.00,
        "theme": "preset:numen"
    }
}
`
	path := write(t, sized)
	if err := settings.Save(path, newDocument(),
		settings.Setting{At: []string{"appearance", "interface_scale"}, Written: 1.25},
		settings.Setting{At: []string{"appearance", "text_scale"}, Written: 1.5}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Replace(sized, `"interface_scale": 1.00`, `"interface_scale": 1.25`, 1)
	want = strings.Replace(want, `"text_scale": 1.00`, `"text_scale": 1.5`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// The size stored under the old name stays where it is, and the setting that
// replaced it is written next to it.
func TestASizeIsSavedAdjacentTheZoomAFileStillNames(t *testing.T) {
	path := write(t, `{
  "appearance": {
    "zoom": 1.5
  }
}
`)
	if err := settings.Save(path, newDocument(),
		settings.Setting{At: []string{"appearance", "interface_scale"}, Written: 1.25}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := `{
  "appearance": {
    "zoom": 1.5,
    "interface_scale": 1.25
  }
}
`
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}

	// Both names in one file: the window is drawn at the one this build reads.
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.25 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
}

// A key missing from the file is appended at the end of its section. The order
// of the existing keys is preserved.
func TestAFieldTheFileHasNotGotIsAppendedToItsSection(t *testing.T) {
	path := write(t, `{
  "appearance": {
    "zoom": 1.5
  },
  "agent": {"use": "claude"}
}
`)
	if err := settings.Save(path, newDocument(), theme("mine:dracula")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := `{
  "appearance": {
    "zoom": 1.5,
    "theme": "mine:dracula"
  },
  "agent": {"use": "claude"}
}
`
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// A section missing from the file is created at the end, indented like the
// key it belongs to.
func TestASectionTheFileHasNotGotIsMadeAtTheEnd(t *testing.T) {
	path := write(t, `{
  "v": 1,
  "agent": {"use": "claude"}
}
`)
	if err := settings.Save(path, newDocument(), theme("mine:dracula")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := `{
  "v": 1,
  "agent": {"use": "claude"},
  "appearance": {
    "theme": "mine:dracula"
  }
}
`
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// A file written on one line takes another field on that line.
func TestAFileOnOneLineStaysOnOneLine(t *testing.T) {
	path := write(t, `{"appearance": {"zoom": 1.5}}`)
	if err := settings.Save(path, newDocument(), theme("mine:dracula")); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"appearance": {"zoom": 1.5, "theme": "mine:dracula"}}`; string(raw) != want {
		t.Errorf("the file came back as %s, and not as %s", raw, want)
	}
}

func TestSave_PreservesUnknownFields(t *testing.T) {
	path := write(t, `{"v":1,"appearance":{"zoom":1.5,"tint":"warm"},"tomorrow":{"of":["a","later","build"]}}`)
	if err := settings.Save(path, newDocument(), theme("preset:nord")); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		Appearance struct {
			Zoom  float64 `json:"zoom"`
			Tint  string  `json:"tint"`
			Theme string  `json:"theme"`
		} `json:"appearance"`
		Tomorrow struct {
			Of []string `json:"of"`
		} `json:"tomorrow"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if file.Appearance.Theme != "preset:nord" {
		t.Errorf("wears %q", file.Appearance.Theme)
	}
	if file.Appearance.Zoom != 1.5 || file.Appearance.Tint != "warm" {
		t.Errorf("the window section is now %+v", file.Appearance)
	}
	if strings.Join(file.Tomorrow.Of, " ") != "a later build" {
		t.Errorf("the section this build does not read is now %+v", file.Tomorrow)
	}
}

// A file with a syntax error is not patched, so the user's content stays as written.
func TestAFileThatDoesNotParseIsNotPatched(t *testing.T) {
	body := `{"appearance": {"zoom": 1.5,,}}`
	path := write(t, body)
	if err := settings.Save(path, newDocument(), theme("preset:nord")); err == nil {
		t.Fatal("a file nothing can read took a setting")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != body {
		t.Errorf("the file is now:\n%s", raw)
	}
}

// A settings file naming one section is a valid settings file, so a patch of a
// file that is not there writes the setting and nothing else.
func TestSavingIntoAFileThatIsNotThereWritesTheSettingAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	if err := settings.Save(path, newDocument(), theme("preset:gruvbox")); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var file map[string]any
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if len(file) != 1 {
		t.Errorf("the file holds more than the setting:\n%s", raw)
	}

	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Theme != "preset:gruvbox" {
		t.Errorf("wears %q", cfg.Appearance.Theme)
	}
	if cfg.Appearance.Mode != settings.ModeSystem || cfg.Appearance.TextScale != settings.AsDesigned {
		t.Errorf("the rest of the file is not the defaults: %+v", cfg)
	}
}

func TestTheSettingsAreLeftReadableByThePersonAlone(t *testing.T) {
	path := write(t, `{"appearance":{"zoom":1.5}}`)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(path, newDocument(), theme("preset:nord")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows keeps no mode on a file: what it answers is the read-only
	// attribute drawn as one.
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("the file is %v", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("the folder holds %d files", len(entries))
	}
}

// Keeping the settings in a dotfiles repository and linking them into place is
// an ordinary arrangement. The link is where the settings are reached, and the
// file it leads to is where they are written.
func TestASaveLandsOnTheFileALinkLeadsTo(t *testing.T) {
	for what, exists := range map[string]bool{
		"a link to a file that is there":   true,
		"a link to a file that is not yet": false,
	} {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "dotfiles", "numen.json")
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if exists {
			target := []byte(`{"appearance":{"theme":"preset:numen"}}`)
			if err := os.WriteFile(destPath, target, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		link := filepath.Join(dir, "numen.json")
		if err := os.Symlink(destPath, link); err != nil {
			t.Fatal(err)
		}

		if err := settings.Save(link, newDocument(), theme("preset:nord")); err != nil {
			t.Fatalf("%s: %v", what, err)
		}

		if info, err := os.Lstat(link); err != nil {
			t.Fatal(err)
		} else if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s: the link is now a file of its own", what)
		}
		raw, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if !strings.Contains(string(raw), "preset:nord") {
			t.Errorf("%s: the file the link leads to holds:\n%s", what, raw)
		}
	}
}

// The settings a launch writes down follow the same link a save does, so the
// bytes land in the file the link leads to and the link stays a link.
func TestAnUntouchedInstallationIsSavedThroughTheLink(t *testing.T) {
	dir := t.TempDir()
	destPath := filepath.Join(dir, "dotfiles", "numen.json")
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "numen.json")
	if err := os.Symlink(destPath, link); err != nil {
		t.Fatal(err)
	}

	if _, err := openAt(link); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the link is now a file of its own")
	}
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("the file the link leads to is not there: %v", err)
	}
}

// A size already present in the file is left for the user to correct, and every
// other setting remains saveable.
func TestASizeAlreadyInTheFileDoesNotBlockTheRest(t *testing.T) {
	const target = `{"appearance":{"text_scale":9,"theme":"preset:numen"}}`
	path := write(t, target)

	if err := settings.Save(path, newDocument(), theme("preset:nord")); err != nil {
		t.Fatalf("a theme could not be saved beside a size out of its band: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(target, `"preset:numen"`, `"preset:nord"`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}

	err = settings.Save(path, newDocument(), settings.Setting{At: []string{"appearance", "text_scale"}, Written: 9})
	if err == nil {
		t.Fatal("a size out of its band was written")
	}
	if !strings.Contains(err.Error(), "appearance.text_scale") {
		t.Errorf("said %q, wanted it to name the setting handed in", err)
	}
}

// A setting is handed in at whatever name it is written under, and a section
// written whole carries the numbers inside it.
func TestASectionHandedInIsCheckedAtTheNumbersItContains(t *testing.T) {
	path := write(t, `{"appearance":{"theme":"preset:numen"}}`)
	err := settings.Save(path, newDocument(), settings.Setting{
		At:      []string{"appearance"},
		Written: map[string]any{"theme": "preset:nord", "text_scale": 9},
	})
	if err == nil {
		t.Fatal("a section carrying a size out of its band was written")
	}
	if !strings.Contains(err.Error(), "appearance.text_scale") {
		t.Errorf("said %q, wanted it to name the setting", err)
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != `{"appearance":{"theme":"preset:numen"}}` {
		t.Errorf("the file is now:\n%s", raw)
	}
}

// A name written twice is read from the last of the two and patched at the
func TestASectionContainsOneOfAName(t *testing.T) {
	for what, body := range map[string]string{
		"the field being written": `{"appearance":{"theme":"preset:numen","theme":"preset:dracula"}}`,
		"a field beside it":       `{"appearance":{"mode":"dark","mode":"light","theme":"preset:numen"}}`,
		"a section":               `{"appearance":{"theme":"preset:numen"},"appearance":{"mode":"dark"}}`,
	} {
		path := write(t, body)
		err := settings.Save(path, newDocument(), theme("preset:nord"))
		if err == nil {
			t.Errorf("%s: a repeated name took a setting", what)
		} else if !strings.Contains(err.Error(), "one of a name") {
			t.Errorf("%s: said %q, wanted it to name the trouble", what, err)
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if string(raw) != body {
			t.Errorf("%s: the file is now:\n%s", what, raw)
		}
	}
}

func TestASettingGoesInsideASection(t *testing.T) {
	path := write(t, `{"appearance":"warm"}`)
	if err := settings.Save(path, newDocument(), theme("preset:nord")); err == nil {
		t.Fatal("a section that is a string took a field")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"appearance":"warm"}` {
		t.Errorf("the file is now:\n%s", raw)
	}
}

func TestSaveKeepsBothOfTwoConcurrentSaves(t *testing.T) {
	for range 50 {
		path := write(t, arranged)
		mode := settings.Setting{At: []string{"appearance", "mode"}, Written: "dark"}
		results := make(chan error, 2)
		go func() { results <- settings.Save(path, newDocument(), theme("mine:dracula")) }()
		go func() { results <- settings.Save(path, newDocument(), mode) }()
		for range 2 {
			if err := <-results; err != nil {
				t.Fatal(err)
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), `"theme": "mine:dracula"`) || !strings.Contains(string(raw), `"mode": "dark"`) {
			t.Fatalf("one save was lost:\n%s", raw)
		}
	}
}
