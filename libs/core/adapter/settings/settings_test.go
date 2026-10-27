package settings_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// A file naming no size is drawn at what the desktop asks for, so the session
// these tests run in is asked nothing.
func TestMain(m *testing.M) {
	os.Unsetenv("GDK_DPI_SCALE")
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
	return container.Config{SettingsPath: path}.Settings()
}

// newDocument is the whole file as the installation declares it, holding what
// an installation nobody has configured does.
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
	// Both sizes are as designed, so an installation nobody has sized draws
	// every length at the number it was drawn with.
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
		t.Errorf("wears %q", cfg.Appearance.Theme)
	}
}

// The window's settings are four fields of one section, and a file writing one
// of them says nothing about the rest.
func TestAFileNamingTheTextSizeStillWearsTheDefaultTheme(t *testing.T) {
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

// A number the setting does not take stops the settings being read. Nothing is
// drawn at a number nobody asked for, and the file says what a person wrote.
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
	if outside.Least != settings.TextScaleBounds.Least || outside.Most != settings.TextScaleBounds.Most {
		t.Errorf("said the setting goes from %v to %v", outside.Least, outside.Most)
	}
	// What is wrong is the number and how far the setting goes, in the sentence
	// a person is shown.
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

// A window drawn at 1.5 goes on being drawn at 1.5. The two names are one
// setting, and the field is given the name this build reads.
func TestAFileNamingTheZoomIsGivenTheNameThatReplacedIt(t *testing.T) {
	path := write(t, `{"appearance":{"zoom":1.5}}`)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.5 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	// The text is untouched by it: the window is larger and a note is set as it
	// was.
	if cfg.Appearance.TextScale != 1 {
		t.Errorf("text is set at %v", cfg.Appearance.TextScale)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"appearance":{"interface_scale":1.5}}`; string(raw) != want {
		t.Errorf("the file is now %s, and not %s", raw, want)
	}
	// Nothing to tell a person: the window is the size it was, and the file
	// says what this build reads.
	if len(cfg.Notices) != 0 {
		t.Errorf("the person is told %q", cfg.Notices)
	}
}

// The file a person arranged comes back from the renaming with one word of it
// changed, down to the blank lines and a number written to two places.
func TestTheRenamingPreservesEveryOtherByteOfTheFileWhereItWas(t *testing.T) {
	path := write(t, arranged)
	if _, err := openAt(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := strings.Replace(arranged, `"zoom": 1.5`, `"interface_scale": 1.5`, 1)
	if string(raw) != want {
		t.Errorf("the file came back as:\n%s\nand not as:\n%s", raw, want)
	}
}

// The renaming happens once. Every launch after it reads a file naming the
// setting, so there is nothing to carry and nothing to say, ever again.
func TestASecondLaunchRenamesNothingAndReturnsNothing(t *testing.T) {
	path := write(t, arranged)
	if _, err := openAt(path); err != nil {
		t.Fatal(err)
	}
	renamed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(renamed), `"zoom"`) {
		t.Fatalf("the first launch left the file as:\n%s", renamed)
	}

	for launch := range 3 {
		cfg, err := openAt(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("launch %d is drawn at %v", launch, cfg.Appearance.InterfaceScale)
		}
		if len(cfg.Notices) != 0 {
			t.Errorf("launch %d tells the person %q", launch, cfg.Notices)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) != string(renamed) {
			t.Errorf("launch %d left the file as:\n%s", launch, raw)
		}
	}
}

// A file naming both is drawn at the one this build reads, and keeps both
// names: one section holds one of a name.
func TestAFileNamingBothIsGeneratedAtTheOneThisBuildReads(t *testing.T) {
	const both = `{"appearance":{"zoom":1.5,"interface_scale":1.25}}`
	path := write(t, both)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.25 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != both {
		t.Errorf("the file is now %s", raw)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("the person is told %q", cfg.Notices)
	}
}

// A file nothing may be written over is a window that opens, drawn at the
// size the file names under the name it names it by.
func TestAFileThatCannotBeSavedIsReadAndGeneratedAtTheSizeItNames(t *testing.T) {
	folder := t.TempDir()
	path := filepath.Join(folder, "numen.json")
	if err := os.WriteFile(path, []byte(arranged), 0o600); err != nil {
		t.Fatal(err)
	}
	testsupport.MakeUnwritable(t, path)

	for launch := range 2 {
		cfg, err := openAt(path)
		if err != nil {
			t.Fatalf("launch %d: %v", launch, err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("launch %d is drawn at %v", launch, cfg.Appearance.InterfaceScale)
		}
		if len(cfg.Notices) != 0 {
			t.Errorf("launch %d tells the person %q", launch, cfg.Notices)
		}
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != arranged {
		t.Errorf("the file is now:\n%s", raw)
	}
}

// Zero named no size, and every file this application has ever written for
// itself holds one.
func TestAZoomNamingNoSizeCarriesNothingAndReturnsNothing(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"zoom":0}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	if len(cfg.Notices) != 0 {
		t.Errorf("the person is told %q", cfg.Notices)
	}
}

// A session that scaled its own text goes on being drawn by it, whether the
// file is empty about size or names the size of the text alone.
func TestAFileNamingNoSizeIsGeneratedAtWhatTheDesktopAsksFor(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "1.5")
	for _, body := range []string{`{}`, `{"appearance":{"zoom":0}}`, `{"appearance":{"text_scale":1.25}}`} {
		cfg, err := openAt(write(t, body))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("%s is drawn at %v", body, cfg.Appearance.InterfaceScale)
		}
	}
}

// A file naming a size is drawn at it, under either name.
func TestWhatTheDesktopAsksForIsNotReadOverTheFile(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "1.5")
	for body, scale := range map[string]float64{
		`{"appearance":{"interface_scale":1.25}}`: 1.25,
		`{"appearance":{"zoom":1.25}}`:            1.25,
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

// A number the setting does not take is not carried onto it, and a window whose
// file was written by an older build opens. The field keeps the name it has, so
// the launch after this one is not rejected over a number a person never wrote
// under that name.
func TestAZoomOutsideWhatTheSizeGoesToIsNotCarried(t *testing.T) {
	const target = `{"appearance":{"zoom":3}}`
	path := write(t, target)
	cfg, err := openAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != target {
		t.Errorf("the file is now %s", raw)
	}

	// something about: the field, the setting it would be read as, and how far
	// that setting goes.
	if len(cfg.Notices) != 1 {
		t.Fatalf("the person is told %q", cfg.Notices)
	}
	for _, about := range []string{"appearance.zoom", "interface_scale", "0.8", "2"} {
		if !strings.Contains(cfg.Notices[0], about) {
			t.Errorf("%q says nothing about %s", cfg.Notices[0], about)
		}
	}
}

// The line is shown on one line of a band, which gives it about sixty
// characters and cuts what is past that off the end. The end is where what a
// person can do about it is written, so the line is one a number cannot
// lengthen: every number a setting refuses says the same thing.
func TestWhatIsSaidFitsTheLineItIsShownOn(t *testing.T) {
	// Numbers a file can hold and this setting will not take, written the ways
	// a person and a machine write them.
	for _, zoom := range []string{
		"3",
		"2.0001",
		"3.141592653589793",
		"0.7999999999999999",
		"123456789.12",
		"1e308",
		"1e-300",
		"0.0000000000000000000000001",
		"12345678901234567890123456789",
	} {
		cfg, err := openAt(write(t, `{"appearance":{"zoom":`+zoom+`}}`))
		if err != nil {
			t.Fatalf("%s: %v", zoom, err)
		}
		if len(cfg.Notices) != 1 {
			t.Fatalf("%s: the person is told %q", zoom, cfg.Notices)
		}
		if response := cfg.Notices[0]; len(response) > 62 {
			t.Errorf("%s: %d characters, cut at 62: %q", zoom, len(response), response)
		}
	}
}

// What the file holds is what is written back, and the two names are not both
// written.
func TestWhatIsSavedBackNamesTheSettingThisBuildReads(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"zoom":1.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"zoom"`) || !strings.Contains(string(raw), `"interface_scale":1.5`) {
		t.Errorf("written back as %s", raw)
	}
	if strings.Contains(string(raw), "appearance.zoom") {
		t.Errorf("written back as %s", raw)
	}
}

func TestAWindowIsDressedByWhatTheFileNames(t *testing.T) {
	cfg, err := openAt(write(t, `{"appearance":{"mode":"dark","theme":"mine:dracula"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Mode != settings.ModeDark || cfg.Appearance.Theme != "mine:dracula" {
		t.Errorf("drawn as %+v", cfg.Appearance)
	}
}

// A person who has run the binary and nothing else has a file to read and
// change, holding what the application is doing.
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
	// Read back, the file says what the run it was written by was doing.
	if back.Indexing.Embedding != cfg.Indexing.Embedding {
		t.Errorf("the embedding section is %+v, was %+v",
			back.Indexing.Embedding, cfg.Indexing.Embedding)
	}
	// A key nobody set is not a field of the file.
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
	// A file that names the window must leave the index alone. Both are
	// sections of one file, and the section nobody wrote about is unchanged.
	if cfg.Indexing.Embedding != container.DefaultEmbedding() {
		t.Errorf("the embedding section is %+v", cfg.Indexing.Embedding)
	}
}

// A file naming one field of one section leaves everything else alone.
func TestAFileNamingOneFieldKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := openAt(write(t, `{"indexing":{"embedding":{"floor":0.3}}}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.Indexing.Embedding
	if e.Floor != 0.3 {
		t.Errorf("the floor is %v", e.Floor)
	}
	if e.MaxTokens != container.DefaultEmbedding().MaxTokens {
		t.Errorf("a chunk is cut at %d tokens", e.MaxTokens)
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
	cfg, err := openAt(write(t, `{"agent":{"use":"","claude":{"model":"opus","max_steps":12}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent.IsNamed() || cfg.Agent.GetUse() != "" {
		t.Errorf("uses %q", cfg.Agent.GetUse())
	}
	if got := cfg.Agent.GetProgram("claude"); got.Model != "opus" || got.MaxSteps != 12 {
		t.Errorf("got %+v", got)
	}
}

// The number in the file is the number of steps. Nothing writes a nought there
// and calls it a ceiling.
func TestAnUntouchedInstallationCarriesAStepCount(t *testing.T) {
	cfg, err := openAt(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Agent.IsNamed() || cfg.Agent.GetUse() != "" {
		t.Errorf("the choice of program is left to the default, and the settings name %q", cfg.Agent.GetUse())
	}
	if got := cfg.Agent.GetProgram("claude"); got.MaxSteps <= 0 {
		t.Errorf("max steps is %d", got.MaxSteps)
	}
	if cfg.Agent.GetProgram("claude").Options["reads_hooks_and_skills"] == "true" {
		t.Error("reads what this machine holds for it")
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
	if got.MaxSteps != settings.DefaultAgentSteps {
		t.Errorf("max steps is %d", got.MaxSteps)
	}
}

// A setting that is on where a file says nothing is read through a pointer:
// leaving the section out, leaving the field out and writing false are three
// different things a person can write, and the last of them is the only one
// that turns it off.
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

// An installation nobody has configured is written down as what it is doing, so
// the setting a person turns off is in front of them.
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
