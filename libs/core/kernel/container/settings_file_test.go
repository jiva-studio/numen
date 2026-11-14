package container_test

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
)

// A file without a size uses the desktop's scale; unsetting GDK_DPI_SCALE keeps
// the tests independent of the session environment.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("GDK_DPI_SCALE"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func writeSettingsFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func openSettingsAt(path string) (settings.File, error) {
	return container.NewConfig(container.Config{
		Paths: container.Paths{SettingsPath: path},
	}).Settings()
}

func TestAnUntouchedInstallationIsGeneratedAsDesigned(t *testing.T) {
	cfg, err := openSettingsAt(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Both scales default to 1, so an unconfigured installation renders every
	// length at its designed size.
	if cfg.Appearance.InterfaceScale != 1 || cfg.Appearance.TextScale != 1 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("a file nobody wrote says %q", cfg.Notices)
	}
	if cfg.Appearance.Mode != theme.SchemeSystem {
		t.Errorf("the colors are read as %q", cfg.Appearance.Mode)
	}
	if cfg.Appearance.Theme != settings.DefaultTheme {
		t.Errorf("applied %q", cfg.Appearance.Theme)
	}
}

// The appearance settings are four fields of one section; a file that sets one
// of them leaves the others at their defaults.
func TestOpen_PartialAppearanceKeepsDefaults(t *testing.T) {
	cfg, err := openSettingsAt(writeSettingsFile(t, `{"app":{"appearance":{"textScale":1.5}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.TextScale != 1.5 || cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
	if cfg.Appearance.Mode != theme.SchemeSystem || cfg.Appearance.Theme != settings.DefaultTheme {
		t.Errorf("drawn as %+v", cfg.Appearance)
	}
}

// A number outside the setting's range is not applied, and the file is still read.
func TestASizeOutsideWhatItGoesToIsReadAndNotApplied(t *testing.T) {
	for body, key := range map[string]string{
		`{"app":{"appearance":{"interfaceScale":2.5}}}`: "interfaceScale",
		`{"app":{"appearance":{"interfaceScale":0.5}}}`: "interfaceScale",
		`{"app":{"appearance":{"interfaceScale":0}}}`:   "interfaceScale",
		`{"app":{"appearance":{"textScale":1.9}}}`:      "textScale",
		`{"app":{"appearance":{"textScale":0}}}`:        "textScale",
	} {
		cfg, err := openSettingsAt(writeSettingsFile(t, body))
		if err != nil {
			t.Errorf("%s was refused: %v", body, err)
			continue
		}
		if got, _ := cfg.GetValue("app", "appearance", key); got != 1.0 {
			t.Errorf("%s: %s in force is %v, want the default", body, key, got)
		}
		if !hasSettingsNotice(cfg.Notices, "appearance."+key) {
			t.Errorf("%s: the notices are %q", body, cfg.Notices)
		}
	}
}

func TestASizeAtEitherEndIsTaken(t *testing.T) {
	cfg, err := openSettingsAt(writeSettingsFile(t, `{"app":{"appearance":{"interfaceScale":0.8,"textScale":1.75}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 0.8 || cfg.Appearance.TextScale != 1.75 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
}

// A session that scaled its own text goes on being drawn by it, whether the
// file is empty about size or names the size of the text alone.
func TestAFileNamingNoSizeIsGeneratedAtWhatTheDesktopAsksFor(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "1.5")
	for _, body := range []string{`{}`, `{"app":{"appearance":{"textScale":1.25}}}`} {
		cfg, err := openSettingsAt(writeSettingsFile(t, body))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("%s is drawn at %v", body, cfg.Appearance.InterfaceScale)
		}
	}
}

// A file naming a size is drawn at it.
func TestWhatTheDesktopAsksForIsNotReadOverTheFile(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "1.5")
	for body, scale := range map[string]float64{
		`{"app":{"appearance":{"interfaceScale":1.25}}}`: 1.25,
	} {
		cfg, err := openSettingsAt(writeSettingsFile(t, body))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != scale {
			t.Errorf("%s is drawn at %v", body, cfg.Appearance.InterfaceScale)
		}
	}
}

// A number the setting does not accept is not one it is seeded with.
func TestADesktopScaleOutsideWhatTheSizeGoesToIsNotTaken(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "3")
	cfg, err := openSettingsAt(writeSettingsFile(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
}

func TestAWindowAppliesTheThemeTheFileNames(t *testing.T) {
	cfg, err := openSettingsAt(writeSettingsFile(t, `{"app":{"appearance":{"mode":"dark","theme":"mine:dracula"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Mode != theme.SchemeDark || cfg.Appearance.Theme != "mine:dracula" {
		t.Errorf("drawn as %+v", cfg.Appearance)
	}
}

// A first run with no other configuration has a settings file to read and edit,
// containing the application's active settings.
func TestAnUnconfiguredInstallationWritesItsSettingsDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	cfg, err := openSettingsAt(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back settings.File
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	// Read back, the file contains the settings of the run that wrote it.
	if back.Appearance != cfg.Appearance {
		t.Errorf("the appearance section is %+v, was %+v", back.Appearance, cfg.Appearance)
	}
	// An unset key is not a field of the file.
	if strings.Contains(string(raw), `"key"`) {
		t.Errorf("the file offers a key:\n%s", raw)
	}
}

func TestOneSettingIsAValidFile(t *testing.T) {
	cfg, err := openSettingsAt(writeSettingsFile(t, `{"app":{"appearance":{"interfaceScale":1.5}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.5 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	// A file that sets the appearance leaves the performance section unchanged. Both are sections
	// of one file, and a section absent from the file keeps its defaults.
	if cfg.Performance != container.DefaultSettings().Performance {
		t.Errorf("the performance section is %+v", cfg.Performance)
	}
}

// An older file naming key_env loads, and no variable of the environment, the
// one it names or any other, becomes a key of the settings.
func TestNoEnvironmentVariableProducesTheServiceAKey(t *testing.T) {
	variables := map[string]string{
		"NUMEN_EMBEDDING_KEY":    "sk-env-numen-embedding",
		"NUMEN_PROOFREADING_KEY": "sk-env-numen-proofreading",
		"OPENAI_API_KEY":         "sk-env-openai",
		"OPENROUTER_API_KEY":     "sk-env-openrouter",
		"MY_OWN_KEY":             "sk-env-my-own",
	}
	for name, value := range variables {
		t.Setenv(name, value)
	}

	cfg, err := openSettingsAt(writeSettingsFile(t, `{"indexing":{"embedding":{
		"indexing":{"use":"service","service":{"name":"a-model","key_env":"MY_OWN_KEY"}},
		"query":{"use":"service","service":{"name":"a-model","key_env":"OPENROUTER_API_KEY"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "key_env") {
		t.Errorf("the settings name a variable: %s", raw)
	}
	for name, value := range variables {
		if strings.Contains(string(raw), value) {
			t.Errorf("the key in %s is in %s", name, raw)
		}
	}
}

func TestWritingTheSettingsBackDoesNotCarryTheKey(t *testing.T) {
	cfg, err := openSettingsAt(writeSettingsFile(t, `{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-secret"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-secret") {
		t.Errorf("the key is in %s", raw)
	}
}

func TestBrokenJSONIsAnError(t *testing.T) {
	if _, err := openSettingsAt(writeSettingsFile(t, `{"app":{"appearance":`)); err == nil {
		t.Error("want an error")
	}
}

func TestThereIsOneFileAndItIsNamedForWhatItIs(t *testing.T) {
	path, err := settings.Path()
	if err != nil {
		t.Fatal(err)
	}
	// Named after the application. `config.json` is a vault's own identity, and
	// one word for two things is one word too few.
	if filepath.Base(filepath.Dir(path)) != "numen" || filepath.Base(path) != "numen.json" {
		t.Errorf("got %q", path)
	}
}

// A setting that defaults to on is read through a pointer: omitting the section,
// omitting the field and writing false are three distinct inputs, and only the
// last turns it off.
func TestATitleAndAFilenameAreOneNameUntilTheFileReturnsOtherwise(t *testing.T) {
	for name, c := range map[string]struct {
		file       string
		shouldSync bool
		wasWritten bool
	}{
		"a file nobody wrote":     {shouldSync: true},
		"a file naming no naming": {file: `{"app":{"appearance":{"textScale":1.5}}}`, shouldSync: true, wasWritten: true},
		"a section naming no field": {
			file: `{"app":{"naming":{}}}`, shouldSync: true, wasWritten: true,
		},
		"a field naming one name": {
			file: `{"app":{"naming":{"syncTitleAndFilename":true}}}`, shouldSync: true, wasWritten: true,
		},
		"a field naming nothing": {
			file: `{"app":{"naming":{"syncTitleAndFilename":null}}}`, shouldSync: true, wasWritten: true,
		},
		"a field telling the two apart": {
			file: `{"app":{"naming":{"syncTitleAndFilename":false}}}`, shouldSync: false, wasWritten: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "numen.json")
			if c.wasWritten {
				path = writeSettingsFile(t, c.file)
			}
			cfg, err := openSettingsAt(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Sync() != c.shouldSync {
				t.Errorf("a title and a filename are one name: %v", cfg.Sync())
			}
		})
	}
}

// An unconfigured installation writes out its active settings, so a setting the
// user can turn off is visible in the file.
func TestAnUntouchedInstallationWritesTheNamingDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	if _, err := openSettingsAt(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"syncTitleAndFilename": true`) {
		t.Errorf("the naming is not in the file it wrote:\n%s", raw)
	}
}

func hasSettingsNotice(notices []string, words ...string) bool {
	for _, notice := range notices {
		found := true
		for _, word := range words {
			found = found && strings.Contains(notice, word)
		}
		if found {
			return true
		}
	}
	return false
}
