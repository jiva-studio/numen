package settings_test

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/container"
)

// A file without a size uses the desktop's scale; unsetting GDK_DPI_SCALE keeps
// the tests independent of the session environment.
func TestMain(m *testing.M) {
	if err := os.Unsetenv("GDK_DPI_SCALE"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "numen.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// openAt is the file as the installation reads it: the sections this adapter
// owns, and the section each adapter is configured by, which is read into that
// adapter's own shape where every adapter is bound.
func openAt(path string) (container.Settings, error) {
	return container.NewConfig(container.Config{SettingsPath: path}).Settings()
}

// newDocument is the complete settings file of an unconfigured installation.
func newDocument() *container.Settings {
	target := container.DefaultSettings()
	return &target
}

func TestAnUntouchedInstallationIsGeneratedAsDesigned(t *testing.T) {
	cfg, err := openAt(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Embedding.MaxTokens == 0 {
		t.Errorf("no chunk bound: %+v", cfg.Indexing.Embedding)
	}
	// Both scales default to 1, so an unconfigured installation renders every
	// length at its designed size.
	if cfg.Appearance.InterfaceScale != 1 || cfg.Appearance.TextScale != 1 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("a file nobody wrote says %q", cfg.Notices)
	}
	if cfg.Appearance.Mode != settings.ModeSystem {
		t.Errorf("the colours are read as %q", cfg.Appearance.Mode)
	}
	if cfg.Appearance.Theme != settings.DefaultTheme {
		t.Errorf("applied %q", cfg.Appearance.Theme)
	}
}

// The appearance settings are four fields of one section; a file that sets one
// of them leaves the others at their defaults.
func TestOpen_PartialAppearanceKeepsDefaults(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"text_scale":1.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.TextScale != 1.5 || cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
	if cfg.Appearance.Mode != settings.ModeSystem || cfg.Appearance.Theme != settings.DefaultTheme {
		t.Errorf("drawn as %+v", cfg.Appearance)
	}
}

// A number outside the setting's range makes reading the settings fail; no
// scale is applied that the file does not contain.
func TestASizeOutsideWhatItGoesToIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"appearance":{"interface_scale":2.5}}`,
		`{"appearance":{"interface_scale":0.5}}`,
		`{"appearance":{"interface_scale":0}}`,
		`{"appearance":{"text_scale":1.9}}`,
		`{"appearance":{"text_scale":0}}`,
	} {
		if _, err := openAt(write(t, body)); err == nil {
			t.Errorf("%s was read", body)
		}
	}

	_, err := openAt(write(t, `{"appearance":{"text_scale":3}}`))
	var outside *settings.OutsideBounds
	if !errors.As(err, &outside) {
		t.Fatalf("failed with %v", err)
	}
	if outside.At != "appearance.text_scale" || outside.Number != 3 {
		t.Errorf("failed %+v", outside)
	}
	if outside.Min != settings.GetTextScaleBounds().Min || outside.Max != settings.GetTextScaleBounds().Max {
		t.Errorf("said the setting goes from %v to %v", outside.Min, outside.Max)
	}
	// The error message states the value and the allowed range.
	if response := outside.Error(); !strings.Contains(response, "appearance.text_scale is 3") {
		t.Errorf("says %q", response)
	}
}

func TestASizeAtEitherEndIsTaken(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"interface_scale":0.8,"text_scale":1.75}}`))
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
	for _, body := range []string{`{}`, `{"appearance":{"text_scale":1.25}}`} {
		cfg, err := openAt(write(t, body))
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
		`{"appearance":{"interface_scale":1.25}}`: 1.25,
	} {
		cfg, err := openAt(write(t, body))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != scale {
			t.Errorf("%s is drawn at %v", body, cfg.Appearance.InterfaceScale)
		}
	}
}

// A number the setting does not take is not one it is seeded with.
func TestADesktopScaleOutsideWhatTheSizeGoesToIsNotTaken(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "3")
	cfg, err := openAt(write(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
}

func TestAWindowAppliesTheThemeTheFileNames(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"mode":"dark","theme":"mine:dracula"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Mode != settings.ModeDark || cfg.Appearance.Theme != "mine:dracula" {
		t.Errorf("drawn as %+v", cfg.Appearance)
	}
}

// A first run with no other configuration has a settings file to read and edit,
// containing the application's active settings.
func TestAnInstallationNobodyConfiguredWritesItsSettingsDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen", "numen.json")
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back container.Settings
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	// Read back, the file contains the settings of the run that wrote it.
	if back.Indexing.Embedding != cfg.Indexing.Embedding {
		t.Errorf("the embedding section is %+v, was %+v",
			back.Indexing.Embedding, cfg.Indexing.Embedding)
	}
	// An unset key is not a field of the file.
	if strings.Contains(string(raw), `"key"`) {
		t.Errorf("the file offers a key:\n%s", raw)
	}
}

func TestOneSettingIsAValidFile(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"interface_scale":1.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.5 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	// A file that sets the appearance leaves the index unchanged. Both are sections
	// of one file, and a section absent from the file keeps its defaults.
	if cfg.Indexing.Embedding != container.DefaultEmbedding() {
		t.Errorf("the embedding section is %+v", cfg.Indexing.Embedding)
	}
}

// A section the file leaves empty keeps its defaults.
func TestAFileNamingAnEmptySectionKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := openAt(write(t, `{"indexing":{"embedding":{}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding; got != container.DefaultEmbedding() {
		t.Errorf("the embedding section is %+v", got)
	}
}

// A file naming the cut of a chunk replaces the default.
func TestAFileNamingTheTokenLimitReplacesTheDefault(t *testing.T) {
	cfg, err := openAt(write(t, `{"indexing":{"embedding":{"max_tokens":128}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.MaxTokens; got != 128 {
		t.Errorf("a chunk is cut at %d tokens", got)
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

	cfg, err := openAt(write(t, `{"indexing":{"embedding":{
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
	cfg, err := openAt(write(t, `{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-secret"}}}}}`))
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
	if _, err := openAt(write(t, `{"appearance":`)); err == nil {
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

// A section is kept whether it is the one in use or not, so trying another agent
// for an afternoon does not cost the settings of the one before.
func TestASectionSurvivesNotBeingTheOneInUse(t *testing.T) {
	cfg, err := openAt(write(t, `{"agent":{"use":"","claude":{"model":"opus"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent.IsNamed() || cfg.Agent.GetUse() != "" {
		t.Errorf("uses %q", cfg.Agent.GetUse())
	}
	if got := cfg.Agent.GetProgram("claude"); got.Model != "opus" {
		t.Errorf("got %+v", got)
	}
}

// An installation the folders looked in do not cover names the command line
// itself, and it is started as written.
func TestTheCommandLineCanBeNamed(t *testing.T) {
	cfg, err := openAt(write(t, `{"agent":{"claude":{"command":["/opt/claude/bin/claude"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agent.GetProgram("claude").Command; len(got) != 1 || got[0] != "/opt/claude/bin/claude" {
		t.Errorf("got %q", got)
	}
}

// One field named leaves the rest of its section alone.
func TestOneAgentFieldKeepsTheRest(t *testing.T) {
	cfg, err := openAt(write(t, `{"agent":{"claude":{"reads_hooks_and_skills":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.Agent.GetProgram("claude")
	if got.Options["reads_hooks_and_skills"] != "true" {
		t.Error("what the file said was not read")
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
		"a file naming no naming": {file: `{"appearance":{"text_scale":1.5}}`, shouldSync: true, wasWritten: true},
		"a section naming no field": {
			file: `{"naming":{}}`, shouldSync: true, wasWritten: true,
		},
		"a field naming one name": {
			file: `{"naming":{"sync_title_and_filename":true}}`, shouldSync: true, wasWritten: true,
		},
		"a field naming nothing": {
			file: `{"naming":{"sync_title_and_filename":null}}`, shouldSync: true, wasWritten: true,
		},
		"a field telling the two apart": {
			file: `{"naming":{"sync_title_and_filename":false}}`, shouldSync: false, wasWritten: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "numen.json")
			if c.wasWritten {
				path = write(t, c.file)
			}
			cfg, err := openAt(path)
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
	if _, err := openAt(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"sync_title_and_filename": true`) {
		t.Errorf("the naming is not in the file it wrote:\n%s", raw)
	}
}
