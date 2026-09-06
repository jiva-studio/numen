package settings_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
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

func TestAnUntouchedInstallationEmbedsLocallyAndIsDrawnAsDesigned(t *testing.T) {
	cfg, err := settings.At(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Embedding.Indexing.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Indexing.Use)
	}
	local, _ := cfg.Indexing.Embedding.Indexing.Local()
	if local.Name == "" || cfg.Indexing.Embedding.Model.Dimensions == 0 {
		t.Errorf("no model to run: %+v", cfg.Indexing.Embedding)
	}
	// Both sizes are as designed, so an installation nobody has sized draws
	// every length at the number it was drawn with.
	if cfg.Appearance.InterfaceScale != 1 || cfg.Appearance.TextScale != 1 {
		t.Errorf("drawn at %+v", cfg.Appearance)
	}
	if len(cfg.Said) != 0 {
		t.Errorf("a file nobody wrote says %q", cfg.Said)
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
	cfg, err := settings.At(write(t, `{"appearance":{"text_scale":1.5}}`))
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
		if _, err := settings.At(write(t, body)); err == nil {
			t.Errorf("%s was read", body)
		}
	}

	_, err := settings.At(write(t, `{"appearance":{"text_scale":3}}`))
	var outside *settings.OutsideBounds
	if !errors.As(err, &outside) {
		t.Fatalf("refused with %v", err)
	}
	if outside.At != "appearance.text_scale" || outside.Number != 3 {
		t.Errorf("refused %+v", outside)
	}
	if outside.Least != settings.TextScaleBounds.Least || outside.Most != settings.TextScaleBounds.Most {
		t.Errorf("said the setting goes from %v to %v", outside.Least, outside.Most)
	}
	// What is wrong is the number and how far the setting goes, in the sentence
	// a person is shown.
	if said := outside.Error(); !strings.Contains(said, "appearance.text_scale is 3") {
		t.Errorf("says %q", said)
	}
}

func TestASizeAtEitherEndIsTaken(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"interface_scale":0.8,"text_scale":1.75}}`))
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
	cfg, err := settings.At(path)
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
	if len(cfg.Said) != 0 {
		t.Errorf("the person is told %q", cfg.Said)
	}
}

// The file a person arranged comes back from the renaming with one word of it
// changed, down to the blank lines and a number written to two places.
func TestTheRenamingLeavesEveryOtherByteOfTheFileWhereItWas(t *testing.T) {
	path := write(t, arranged)
	if _, err := settings.At(path); err != nil {
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
func TestASecondLaunchRenamesNothingAndSaysNothing(t *testing.T) {
	path := write(t, arranged)
	if _, err := settings.At(path); err != nil {
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
		cfg, err := settings.At(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("launch %d is drawn at %v", launch, cfg.Appearance.InterfaceScale)
		}
		if len(cfg.Said) != 0 {
			t.Errorf("launch %d tells the person %q", launch, cfg.Said)
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
func TestAFileNamingBothIsDrawnAtTheOneThisBuildReads(t *testing.T) {
	const both = `{"appearance":{"zoom":1.5,"interface_scale":1.25}}`
	path := write(t, both)
	cfg, err := settings.At(path)
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
	if len(cfg.Said) != 0 {
		t.Errorf("the person is told %q", cfg.Said)
	}
}

// A file nothing may be written over is a window that opens, drawn at the
// size the file names under the name it names it by.
func TestAFileThatCannotBeWrittenIsReadAndDrawnAtTheSizeItNames(t *testing.T) {
	folder := t.TempDir()
	path := filepath.Join(folder, "numen.json")
	if err := os.WriteFile(path, []byte(arranged), 0o600); err != nil {
		t.Fatal(err)
	}
	testsupport.Unwritable(t, path)

	for launch := range 2 {
		cfg, err := settings.At(path)
		if err != nil {
			t.Fatalf("launch %d: %v", launch, err)
		}
		if cfg.Appearance.InterfaceScale != 1.5 {
			t.Errorf("launch %d is drawn at %v", launch, cfg.Appearance.InterfaceScale)
		}
		if len(cfg.Said) != 0 {
			t.Errorf("launch %d tells the person %q", launch, cfg.Said)
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
func TestAZoomNamingNoSizeCarriesNothingAndSaysNothing(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"zoom":0}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	if len(cfg.Said) != 0 {
		t.Errorf("the person is told %q", cfg.Said)
	}
}

// A session that scaled its own text goes on being drawn by it, whether the
// file names nothing about size or names the size of the text alone.
func TestAFileNamingNoSizeIsDrawnAtWhatTheDesktopAsksFor(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "1.5")
	for _, body := range []string{`{}`, `{"appearance":{"zoom":0}}`, `{"appearance":{"text_scale":1.25}}`} {
		cfg, err := settings.At(write(t, body))
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
	for body, drawn := range map[string]float64{
		`{"appearance":{"interface_scale":1.25}}`: 1.25,
		`{"appearance":{"zoom":1.25}}`:            1.25,
	} {
		cfg, err := settings.At(write(t, body))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Appearance.InterfaceScale != drawn {
			t.Errorf("%s is drawn at %v", body, cfg.Appearance.InterfaceScale)
		}
	}
}

// A number the setting does not take is not one it is seeded with.
func TestADesktopScaleOutsideWhatTheSizeGoesToIsNotTaken(t *testing.T) {
	t.Setenv("GDK_DPI_SCALE", "3")
	cfg, err := settings.At(write(t, `{}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
}

// A number the setting does not take is not carried onto it, and a window whose
// file was written by an older build opens. The field keeps the name it has, so
// the launch after this one is not refused over a number a person never wrote
// under that name.
func TestAZoomOutsideWhatTheSizeGoesToIsNotCarried(t *testing.T) {
	const held = `{"appearance":{"zoom":3}}`
	path := write(t, held)
	cfg, err := settings.At(path)
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
	if string(raw) != held {
		t.Errorf("the file is now %s", raw)
	}

	// The one thing a person is told, and the only thing here they can do
	// something about: the field, the setting it would be read as, and how far
	// that setting goes.
	if len(cfg.Said) != 1 {
		t.Fatalf("the person is told %q", cfg.Said)
	}
	for _, about := range []string{"appearance.zoom", "interface_scale", "0.8", "2"} {
		if !strings.Contains(cfg.Said[0], about) {
			t.Errorf("%q says nothing about %s", cfg.Said[0], about)
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
		cfg, err := settings.At(write(t, `{"appearance":{"zoom":`+zoom+`}}`))
		if err != nil {
			t.Fatalf("%s: %v", zoom, err)
		}
		if len(cfg.Said) != 1 {
			t.Fatalf("%s: the person is told %q", zoom, cfg.Said)
		}
		if said := cfg.Said[0]; len(said) > 62 {
			t.Errorf("%s: %d characters, cut at 62: %q", zoom, len(said), said)
		}
	}
}

// What the file holds is what is written back, and the two names are not both
// written.
func TestWhatIsWrittenBackNamesTheSettingThisBuildReads(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"zoom":1.5}}`))
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
	// What a person is told is about the file and is not a field of it.
	if strings.Contains(string(raw), "appearance.zoom") {
		t.Errorf("written back as %s", raw)
	}
}

func TestAWindowIsDressedByWhatTheFileNames(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"mode":"dark","theme":"mine:dracula"}}`))
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
	cfg, err := settings.At(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back settings.Config
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	// Read back, the file says what the run it was written by was doing.
	if back.Indexing.Embedding.Model != cfg.Indexing.Embedding.Model {
		t.Errorf("the model is %+v, was %+v", back.Indexing.Embedding.Model, cfg.Indexing.Embedding.Model)
	}
	local, ok := back.Indexing.Embedding.Indexing.Local()
	if !ok || !local.Download {
		t.Error("a machine with no model would fetch none")
	}
	// A key nobody set is not a field of the file.
	if strings.Contains(string(raw), `"key"`) {
		t.Errorf("the file offers a key:\n%s", raw)
	}
}

func TestOneSettingIsAValidFile(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"interface_scale":1.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.InterfaceScale != 1.5 {
		t.Errorf("drawn at %v", cfg.Appearance.InterfaceScale)
	}
	// A file that names the window must leave the embedder alone. Both are
	// sections of one file, and the section nobody wrote about is unchanged.
	held, _ := cfg.Indexing.Embedding.Indexing.Local()
	offered, _ := embed.Defaults().Indexing.Local()
	if held.Name != offered.Name {
		t.Errorf("local model is %q", held.Name)
	}
	if cfg.Indexing.Embedding.Indexing.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Indexing.Use)
	}
}

// asLocal and asService are the halves a provider carries, whichever of them is
// in force. The word is what puts one there, and a test reading the other says
// so before it does.
func asLocal(where embed.Provider) embed.LocalModel {
	where.Use = embed.UseLocal
	m, _ := where.Local()
	return m
}

func asService(where embed.Provider) embed.ServiceModel {
	where.Use = embed.UseService
	m, _ := where.Service()
	return m
}

// A file naming one field of one section leaves everything else alone.
func TestAFileNamingOneFieldKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"embedding":{"model":{"name":"text-embedding-3-large","dimensions":3072},
		 "indexing":{"use":"service","service":{"name":"text-embedding-3-large"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.Indexing.Embedding
	if e.Indexing.Use != embed.UseService {
		t.Errorf("uses %q", e.Indexing.Use)
	}
	if e.Model.Name != "text-embedding-3-large" || e.Model.Dimensions != 3072 {
		t.Errorf("got %+v", e.Model)
	}
	// A field the file says nothing about keeps what the defaults set.
	if e.Model.MaxTokens != embed.Defaults().Model.MaxTokens {
		t.Errorf("the model cuts at %d", e.Model.MaxTokens)
	}
	service, _ := e.Indexing.Service()
	if service.BaseURL != asService(embed.Defaults().Indexing).BaseURL {
		t.Errorf("base URL is %q", service.BaseURL)
	}
	// The half the file is not on keeps the defaults as well, and the word is
	// all that stands between the two.
	if local := asLocal(e.Indexing); local.Name != asLocal(embed.Defaults().Indexing).Name {
		t.Errorf("local model is %q", local.Name)
	}
	// A question is asked the way the vault was indexed.
	if got := e.Asking(); got.Use != embed.UseService {
		t.Errorf("questions are embedded by %+v", got)
	}
}

// A model pooled one way is pooled under one word, whether the file says it or
// leaves it out: two words for one pooling are two keys over one set of
// vectors.
func TestAPoolingLeftOutIsTheOneEveryModelHas(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"embedding":{"model":{"name":"e5","dimensions":384}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Model.Pooling; got != embed.PoolMean {
		t.Errorf("pooled %q", got)
	}
}

func TestAVaultIndexedByAServiceIsAskedOnThisMachine(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{
		"model": {"name":"bge-m3","dimensions":1024,"max_tokens":512,"pooling":"head"},
		"indexing": {"use":"service","service":{"base_url":"https://openrouter.ai/api/v1","name":"baai/bge-m3"}},
		"query":    {"use":"local","local":{"name":"BAAI/bge-m3","download":true}}
	}}}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.Indexing.Embedding
	if e.Model.Name != "bge-m3" || e.Model.Dimensions != 1024 || e.Model.Pooling != embed.PoolHead {
		t.Errorf("the model is %+v", e.Model)
	}
	indexing, byService := e.Indexing.Service()
	if !byService || indexing.Name != "baai/bge-m3" {
		t.Errorf("indexed by %+v", e.Indexing)
	}
	asking, here := e.Asking().Local()
	if !here || asking.Name != "BAAI/bge-m3" {
		t.Errorf("asked by %+v", e.Asking())
	}
	// The section nobody wrote about keeps its default.
	if indexing.BatchCharacters != asService(embed.Defaults().Indexing).BatchCharacters {
		t.Errorf("batch is %d", indexing.BatchCharacters)
	}
}

// A vault searched by its words: nothing fetched, nothing asked of a network.
func TestAnInstallationMayNameNoModelAtAll(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"indexing":{"use":""}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Indexing.Use; got != "" {
		t.Errorf("uses %q", got)
	}
	if got := cfg.Indexing.Embedding.Asking().Use; got != "" {
		t.Errorf("questions are embedded by %q", got)
	}
}

func TestTheKeyComesFromTheFileOrTheEnvironment(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "from-the-environment")

	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"indexing":{"service":{"key":"from-the-file"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := asService(cfg.Indexing.Embedding.Indexing).Key(); got != "from-the-file" {
		t.Errorf("got %q", got)
	}

	cfg, err = settings.At(write(t, `{"indexing":{"embedding":{"indexing":{"service":{}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := asService(cfg.Indexing.Embedding.Indexing).Key(); got != "from-the-environment" {
		t.Errorf("got %q", got)
	}
}

func TestAnInstallationMayNameItsOwnEnvironmentVariable(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "from-openrouter")
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"indexing":{"service":{"key_env":"OPENROUTER_API_KEY"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := asService(cfg.Indexing.Embedding.Indexing).Key(); got != "from-openrouter" {
		t.Errorf("got %q", got)
	}
}

func TestWritingTheSettingsBackDoesNotCarryTheKey(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"indexing":{"service":{"key":"sk-secret"}}}}}`))
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
	if _, err := settings.At(write(t, `{"appearance":`)); err == nil {
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
	cfg, err := settings.At(write(t, `{"agent":{"use":"","claude":{"model":"opus","max_steps":12}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent.Use != "" {
		t.Errorf("uses %q", cfg.Agent.Use)
	}
	if cfg.Agent.Claude.Model != "opus" || cfg.Agent.Claude.MaxSteps != 12 {
		t.Errorf("got %+v", cfg.Agent.Claude)
	}
}

// The number in the file is the number of steps. Nothing writes a nought there
// and calls it a ceiling.
func TestAnUntouchedInstallationCarriesAStepCount(t *testing.T) {
	cfg, err := settings.At(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Agent.Use != agent.UseClaude {
		t.Errorf("answers with %q", cfg.Agent.Use)
	}
	if cfg.Agent.Claude.MaxSteps <= 0 {
		t.Errorf("max steps is %d", cfg.Agent.Claude.MaxSteps)
	}
	if cfg.Agent.Claude.ReadsHooksAndSkills {
		t.Error("reads what this machine holds for it")
	}
}

// An installation the folders looked in do not cover names the command line
// itself, and it is started as written.
func TestTheCommandLineCanBeNamed(t *testing.T) {
	cfg, err := settings.At(write(t, `{"agent":{"claude":{"command":["/opt/claude/bin/claude"]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agent.Claude.Command; len(got) != 1 || got[0] != "/opt/claude/bin/claude" {
		t.Errorf("got %q", got)
	}
}

// One field named leaves the rest of its section alone.
func TestOneAgentFieldKeepsTheRest(t *testing.T) {
	was := settings.Defaults().Agent.Claude.MaxSteps
	cfg, err := settings.At(write(t, `{"agent":{"claude":{"reads_hooks_and_skills":true}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Agent.Claude.ReadsHooksAndSkills {
		t.Error("what the file said was not read")
	}
	if cfg.Agent.Claude.MaxSteps != was {
		t.Errorf("max steps is %d", cfg.Agent.Claude.MaxSteps)
	}
}

// Nothing proofreads a reading unless a person named something to proofread it
// with. This is what an untouched installation does, and a model that rewrites
// a person's books does not arrive by default.
func TestAnUntouchedInstallationProofreadsNothing(t *testing.T) {
	cfg, err := settings.At(filepath.Join(t.TempDir(), "numen.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Proofreading.Named() {
		t.Errorf("proofreads with %+v", cfg.Indexing.Proofreading)
	}
}

// A profile named without a model is a profile naming nothing.
func TestAProofreadingProfileWithoutAModelNamesNothing(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"proofreading":{"profiles":{"openrouter":{"use":"service"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Proofreading.Profiles["openrouter"].Named() {
		t.Error("proofreads with a model nobody named")
	}
}

// A profile is flat: every key stands at its own level, and `use` says which of
// them apply.
func TestEachKindOfProofreadingProfileIsRead(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{"profiles":{
		"openrouter": {"use":"service","name":"google/gemini-2.5-flash","batch_size":25},
		"agent":      {"use":"agent","model":"haiku","batch_size":20,"overlap":2}
	}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	read := cfg.Indexing.Proofreading
	if !read.Named() || len(read.Profiles) != 2 {
		t.Fatalf("got %+v", read)
	}

	service := read.Profiles["openrouter"]
	if !service.Named() || service.Name != "google/gemini-2.5-flash" || service.BatchSize != 25 {
		t.Errorf("the service profile is %+v", service)
	}
	if service.BaseURL != proofreading.ServiceDefaults().BaseURL {
		t.Errorf("base URL is %q", service.BaseURL)
	}

	agent := read.Profiles["agent"]
	if !agent.Named() || agent.Model != "haiku" || agent.BatchSize != 20 || agent.Overlap != 2 {
		t.Errorf("the agent profile is %+v", agent)
	}
}

// A profile may name the command line it is reached through, for an
// installation whose own is not `claude` from the path.
func TestAnAgentProfileMayNameItsCommand(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{"profiles":{
		"agent": {"use":"agent","model":"haiku","command":["/opt/claude","--quiet"]}
	}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	held := cfg.Indexing.Proofreading.Profiles["agent"].Command
	if len(held) != 2 || held[0] != "/opt/claude" || held[1] != "--quiet" {
		t.Errorf("got %q", held)
	}
}

// How far a correction may stand from the line as read stands over the
// profiles: it is a property of the text, and one threshold holds for the
// installation.
func TestTheEditDistanceIsReadFromAboveTheProfiles(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{
		"max_edit_distance": 0.5,
		"profiles":{"openrouter":{"use":"service","name":"a-model"}}
	}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Proofreading.Distance(); got != 0.5 {
		t.Errorf("a correction may stand %v from the line", got)
	}

	silent, err := settings.At(write(t,
		`{"indexing":{"proofreading":{"profiles":{"openrouter":{"use":"service","name":"a-model"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := silent.Indexing.Proofreading.Distance(); got != proofreading.DefaultMaxEditDistance {
		t.Errorf("a file saying nothing gives %v", got)
	}
}

// Which profile puts each kind of reading right, and whether it happens without
// anybody asking, is said where the reading is configured.
func TestEachReadingNamesTheProfileThatPutsItRight(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{
		"recognition":   {"proofread":{"with":"openrouter","automatically":true}},
		"transcription": {"proofread":{"with":"agent","automatically":false}}
	}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Recognition.Proofread; got.Profile != "openrouter" || !got.Automatically {
		t.Errorf("a scan is put right by %+v", got)
	}
	if got := cfg.Indexing.Transcription.Proofread; got.Profile != "agent" || got.Automatically {
		t.Errorf("a transcript is put right by %+v", got)
	}
}

func TestTheProofreadersKeyStaysOutOfWhatIsWrittenBack(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{"profiles":{
		"openrouter": {"use":"service","name":"a-model","key":"sk-proof"}
	}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	held := cfg.Indexing.Proofreading.Profiles["openrouter"]
	if got := held.Key(); got != "sk-proof" {
		t.Errorf("got %q", got)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-proof") {
		t.Errorf("the key is in %s", raw)
	}
	if strings.Contains(held.String(), "sk-proof") {
		t.Errorf("the key is in %s", held)
	}
}

// A setting that is on where a file says nothing is read through a pointer:
// leaving the section out, leaving the field out and writing false are three
// different things a person can write, and the last of them is the only one
// that turns it off.
func TestATitleAndAFilenameAreOneNameUntilTheFileSaysOtherwise(t *testing.T) {
	for name, c := range map[string]struct {
		file  string
		sync  bool
		wrote bool
	}{
		"a file nobody wrote":     {sync: true},
		"a file naming no naming": {file: `{"appearance":{"text_scale":1.5}}`, sync: true, wrote: true},
		"a section naming no field": {
			file: `{"naming":{}}`, sync: true, wrote: true,
		},
		"a field naming one name": {
			file: `{"naming":{"sync_title_and_filename":true}}`, sync: true, wrote: true,
		},
		"a field naming nothing": {
			file: `{"naming":{"sync_title_and_filename":null}}`, sync: true, wrote: true,
		},
		"a field telling the two apart": {
			file: `{"naming":{"sync_title_and_filename":false}}`, sync: false, wrote: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "numen.json")
			if c.wrote {
				path = write(t, c.file)
			}
			cfg, err := settings.At(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Sync() != c.sync {
				t.Errorf("a title and a filename are one name: %v", cfg.Sync())
			}
		})
	}
}

// An installation nobody has configured is written down as what it is doing, so
// the setting a person turns off is in front of them.
func TestAnUntouchedInstallationWritesTheNamingDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	if _, err := settings.At(path); err != nil {
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

// A node hangs the headings of its note where the file says nothing, read
// through a pointer the way the naming is: leaving the section out, leaving the
// field out and writing false are three different things a person can write,
// and the last of them is the only one that turns it off.
func TestANodeHangsThePartsOfItsNoteUntilTheFileSaysOtherwise(t *testing.T) {
	for name, c := range map[string]struct {
		file  string
		hangs bool
		wrote bool
	}{
		"a file nobody wrote":         {hangs: true},
		"a file naming no appearance": {file: `{"naming":{}}`, hangs: true, wrote: true},
		"a section naming no field": {
			file: `{"appearance":{"text_scale":1.5}}`, hangs: true, wrote: true,
		},
		"a field hanging them": {
			file: `{"appearance":{"hang_parts_under_a_node":true}}`, hangs: true, wrote: true,
		},
		"a field naming nothing": {
			file: `{"appearance":{"hang_parts_under_a_node":null}}`, hangs: true, wrote: true,
		},
		"a field leaving the box alone": {
			file: `{"appearance":{"hang_parts_under_a_node":false}}`, hangs: false, wrote: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "numen.json")
			if c.wrote {
				path = write(t, c.file)
			}
			cfg, err := settings.At(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Hangs() != c.hangs {
				t.Errorf("a node hangs the parts of its note: %v", cfg.Hangs())
			}
		})
	}
}

// An installation nobody has configured is written down as what it is doing, so
// the setting a person turns off is in front of them.
func TestAnUntouchedInstallationWritesTheHangingDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "numen.json")
	if _, err := settings.At(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"hang_parts_under_a_node": true`) {
		t.Errorf("the hanging is not in the file it wrote:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"parts_under_a_node": 6`) {
		t.Errorf("the count is not in the file it wrote:\n%s", raw)
	}
}

// How many parts stand under a node at once. A file naming no number stands the
// default of them, and a number the setting takes is the number that stands.
func TestANodeStandsAsManyPartsAsTheFileNames(t *testing.T) {
	for name, c := range map[string]struct {
		file  string
		parts int
		wrote bool
	}{
		"a file nobody wrote":         {parts: settings.DefaultParts},
		"a file naming no appearance": {file: `{"naming":{}}`, parts: 6, wrote: true},
		"a section naming no field": {
			file: `{"appearance":{"text_scale":1.5}}`, parts: 6, wrote: true,
		},
		"a field naming a number": {
			file: `{"appearance":{"parts_under_a_node":3}}`, parts: 3, wrote: true,
		},
		"a field at either end": {
			file: `{"appearance":{"parts_under_a_node":12}}`, parts: 12, wrote: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "numen.json")
			if c.wrote {
				path = write(t, c.file)
			}
			cfg, err := settings.At(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Parts() != c.parts {
				t.Errorf("a node stands %d parts", cfg.Parts())
			}
		})
	}
}

// A count the setting does not take stops the settings being read. Nothing is
// drawn at a number nobody asked for, and the file says what a person wrote.
func TestACountOfPartsOutsideWhatItGoesToIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"appearance":{"parts_under_a_node":0}}`,
		`{"appearance":{"parts_under_a_node":-1}}`,
		`{"appearance":{"parts_under_a_node":13}}`,
	} {
		if _, err := settings.At(write(t, body)); err == nil {
			t.Errorf("%s was read", body)
		}
	}

	_, err := settings.At(write(t, `{"appearance":{"parts_under_a_node":20}}`))
	var outside *settings.OutsideBounds
	if !errors.As(err, &outside) {
		t.Fatalf("refused with %v", err)
	}
	if outside.At != "appearance.parts_under_a_node" || outside.Number != 20 {
		t.Errorf("refused %+v", outside)
	}
	if outside.Least != settings.PartsUnderANodeBounds.Least ||
		outside.Most != settings.PartsUnderANodeBounds.Most {
		t.Errorf("said the setting goes from %v to %v", outside.Least, outside.Most)
	}
	if said := outside.Error(); !strings.Contains(said, "appearance.parts_under_a_node is 20") {
		t.Errorf("says %q", said)
	}
}
