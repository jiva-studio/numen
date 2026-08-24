package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
)

func theme(name string) settings.Setting {
	return settings.Setting{At: []string{"appearance", "theme"}, Value: name}
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

	cfg, err := settings.At(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Indexing.Service.Key(); got != "sk-the-persons-own" {
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

	cfg, err := settings.At(path)
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
	if info.Mode().Perm() != 0o600 {
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
