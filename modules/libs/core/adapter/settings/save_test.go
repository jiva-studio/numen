package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
)

func theme(name string) settings.Setting {
	return settings.Setting{At: []string{"appearance", "theme"}, Written: name}
}

// arranged is a file as somebody wrote it by hand: their order, their
// indentation, their sections spaced out, and a number written to two places.
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

// The one test that would have caught a file being rewritten from a map: what
// comes back is the file, with the bytes of one value replaced and nothing else
// touched.
func TestAHandArrangedFileComesBackWithOneValueChanged(t *testing.T) {
	path := write(t, arranged)
	if err := settings.Save(path, theme("mine:dracula")); err != nil {
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

// Two settings at once, and the file is still the person's arrangement.
func TestTheThemeAndTheModeAreWrittenWithoutMovingAnythingElse(t *testing.T) {
	path := write(t, arranged)
	if err := settings.Save(path, theme("preset:nord"),
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

// The two sizes are written where they sit, and a number a person wrote to two
// places beside them comes back written to two places.
func TestTheTwoSizesAreWrittenWithoutMovingAnythingElse(t *testing.T) {
	const sized = `{
    "appearance": {
        "interface_scale": 1.00,
        "text_scale": 1.00,
        "theme": "preset:numen"
    }
}
`
	path := write(t, sized)
	if err := settings.Save(path,
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

// The size a file names under the name it had is left where the person wrote
// it, and the setting that replaced it is written beside it.
func TestASizeIsWrittenBesideTheZoomAFileStillNames(t *testing.T) {
	path := write(t, `{
  "appearance": {
    "zoom": 1.5
  }
}
`)
	if err := settings.Save(path,
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
	cfg, err := settings.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.25 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
}

// A name nobody has written yet goes at the end of the section it belongs to.
// The order of the rest of it is the person's.
func TestAFieldTheFileHasNotGotIsAppendedToItsSection(t *testing.T) {
	path := write(t, `{
  "appearance": {
    "zoom": 1.5
  },
  "agent": {"use": "claude"}
}
`)
	if err := settings.Save(path, theme("mine:dracula")); err != nil {
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

// A section nobody has written yet is made where it belongs, laid out against
// the name it hangs from.
func TestASectionTheFileHasNotGotIsMadeAtTheEnd(t *testing.T) {
	path := write(t, `{
  "v": 1,
  "agent": {"use": "claude"}
}
`)
	if err := settings.Save(path, theme("mine:dracula")); err != nil {
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
	if err := settings.Save(path, theme("mine:dracula")); err != nil {
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

// The one thing a patch is for. A round trip through Config would come back
// without the key, because the key is never something this application writes.
func TestAKeyAPersonTypedIsStillThereAfterAThemeIsSaved(t *testing.T) {
	path := write(t, `{
  "indexing": {
    "embedding": {
      "indexing": {"use": "service", "service": {"key": "sk-the-persons-own"}}
    }
  }
}`)
	if err := settings.Save(path, theme("mine:dracula")); err != nil {
		t.Fatal(err)
	}

	cfg, err := settings.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	service, ok := cfg.Indexing.Embedding.Indexing.Service()
	if !ok {
		t.Fatal("the vault is no longer indexed by a service")
	}
	if got := service.Key(); got != "sk-the-persons-own" {
		t.Errorf("the key is now %q", got)
	}
	if cfg.Appearance.Theme != "mine:dracula" {
		t.Errorf("wears %q", cfg.Appearance.Theme)
	}
}

func TestAFieldThisBuildKnowsNothingAboutComesThroughAsItWas(t *testing.T) {
	path := write(t, `{"v":1,"appearance":{"zoom":1.5,"tint":"warm"},"tomorrow":{"of":["a","later","build"]}}`)
	if err := settings.Save(path, theme("preset:nord")); err != nil {
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

// Half a person's file is worth more than a saved theme, and what is wrong with
// it is theirs to see.
func TestAFileThatDoesNotParseIsNotPatched(t *testing.T) {
	body := `{"appearance": {"zoom": 1.5,,}}`
	path := write(t, body)
	if err := settings.Save(path, theme("preset:nord")); err == nil {
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
	if err := settings.Save(path, theme("preset:gruvbox")); err != nil {
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

	cfg, err := settings.OpenAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Theme != "preset:gruvbox" {
		t.Errorf("wears %q", cfg.Appearance.Theme)
	}
	if cfg.Appearance.Mode != settings.ModeSystem || cfg.Indexing.Embedding.Model.Dimensions == 0 {
		t.Errorf("the rest of the file is not the defaults: %+v", cfg)
	}
}

func TestTheSettingsAreLeftReadableByThePersonAlone(t *testing.T) {
	path := write(t, `{"appearance":{"zoom":1.5}}`)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(path, theme("preset:nord")); err != nil {
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
	// The temporary file it was written through is not left behind.
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
		kept := filepath.Join(dir, "dotfiles", "numen.json")
		if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
			t.Fatal(err)
		}
		if exists {
			held := []byte(`{"appearance":{"theme":"preset:numen"}}`)
			if err := os.WriteFile(kept, held, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		link := filepath.Join(dir, "numen.json")
		if err := os.Symlink(kept, link); err != nil {
			t.Fatal(err)
		}

		if err := settings.Save(link, theme("preset:nord")); err != nil {
			t.Fatalf("%s: %v", what, err)
		}

		if info, err := os.Lstat(link); err != nil {
			t.Fatal(err)
		} else if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%s: the link is now a file of its own", what)
		}
		raw, err := os.ReadFile(kept)
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
func TestAnUntouchedInstallationIsWrittenThroughTheLink(t *testing.T) {
	dir := t.TempDir()
	kept := filepath.Join(dir, "dotfiles", "numen.json")
	if err := os.MkdirAll(filepath.Dir(kept), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "numen.json")
	if err := os.Symlink(kept, link); err != nil {
		t.Fatal(err)
	}

	if _, err := settings.OpenAt(link); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Lstat(link); err != nil {
		t.Fatal(err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the link is now a file of its own")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("the file the link leads to is not there: %v", err)
	}
}

// A number a setting does not take is refused where it is handed in. One the
// file already held is the person's to put right, and every other setting stays
// reachable while they do.
func TestASizeAlreadyInTheFileDoesNotBlockTheRest(t *testing.T) {
	const held = `{"appearance":{"text_scale":9,"theme":"preset:numen"}}`
	path := write(t, held)

	if err := settings.Save(path, theme("preset:nord")); err != nil {
		t.Fatalf("a theme could not be saved beside a size out of its band: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(held, `"preset:numen"`, `"preset:nord"`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}

	err = settings.Save(path, settings.Setting{At: []string{"appearance", "text_scale"}, Written: 9})
	if err == nil {
		t.Fatal("a size out of its band was written")
	}
	if !strings.Contains(err.Error(), "appearance.text_scale") {
		t.Errorf("said %q, wanted it to name the setting handed in", err)
	}
}

// A setting is handed in at whatever name it is written under, and a section
// written whole carries the numbers inside it.
func TestASectionHandedInIsCheckedAtTheNumbersItHolds(t *testing.T) {
	path := write(t, `{"appearance":{"theme":"preset:numen"}}`)
	err := settings.Save(path, settings.Setting{
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
// first. One section holds one of a name, and a file holding two is refused.
func TestASectionHoldsOneOfAName(t *testing.T) {
	for what, body := range map[string]string{
		"the field being written": `{"appearance":{"theme":"preset:numen","theme":"preset:dracula"}}`,
		"a field beside it":       `{"appearance":{"mode":"dark","mode":"light","theme":"preset:numen"}}`,
		"a section":               `{"appearance":{"theme":"preset:numen"},"appearance":{"mode":"dark"}}`,
	} {
		path := write(t, body)
		err := settings.Save(path, theme("preset:nord"))
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
	if err := settings.Save(path, theme("preset:nord")); err == nil {
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
