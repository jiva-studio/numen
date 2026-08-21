package settings_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/agent"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/proofreading"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
)

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
	if cfg.Indexing.Embedding.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Use)
	}
	if cfg.Indexing.Embedding.Local.Name == "" || cfg.Indexing.Embedding.Local.Dimensions == 0 {
		t.Errorf("no model to run: %+v", cfg.Indexing.Embedding.Local)
	}
	// Zero is what says the desktop decides, so nothing may fill it in.
	if cfg.Appearance.Zoom != 0 {
		t.Errorf("zoom is %v", cfg.Appearance.Zoom)
	}
}

func TestOneSettingIsAValidFile(t *testing.T) {
	cfg, err := settings.At(write(t, `{"appearance":{"zoom":1.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appearance.Zoom != 1.5 {
		t.Errorf("zoom is %v", cfg.Appearance.Zoom)
	}
	// A file that names the window must leave the embedder alone. Both are
	// sections of one file, and the section nobody wrote about is unchanged.
	if cfg.Indexing.Embedding.Local.Name != embed.Defaults().Local.Name {
		t.Errorf("local model is %q", cfg.Indexing.Embedding.Local.Name)
	}
	if cfg.Indexing.Embedding.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Use)
	}
}

func TestAFileNamingOneFieldKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"embedding":{"use":"service","service":{"name":"text-embedding-3-large","dimensions":3072}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Embedding.Use != embed.UseService {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Use)
	}
	if cfg.Indexing.Embedding.Service.Name != "text-embedding-3-large" || cfg.Indexing.Embedding.Service.Dimensions != 3072 {
		t.Errorf("got %+v", cfg.Indexing.Embedding.Service)
	}
	if cfg.Indexing.Embedding.Service.BaseURL != embed.Defaults().Service.BaseURL {
		t.Errorf("base URL is %q", cfg.Indexing.Embedding.Service.BaseURL)
	}
	if cfg.Indexing.Embedding.Local.Name != embed.Defaults().Local.Name {
		t.Errorf("local model is %q", cfg.Indexing.Embedding.Local.Name)
	}
}

func TestTheKeyComesFromTheFileOrTheEnvironment(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "from-the-environment")

	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"service":{"key":"from-the-file"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Service.Key(); got != "from-the-file" {
		t.Errorf("got %q", got)
	}

	cfg, err = settings.At(write(t, `{"indexing":{"embedding":{"service":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Service.Key(); got != "from-the-environment" {
		t.Errorf("got %q", got)
	}
}

func TestAnInstallationMayNameItsOwnEnvironmentVariable(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "from-openrouter")
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"service":{"key_env":"OPENROUTER_API_KEY"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Service.Key(); got != "from-openrouter" {
		t.Errorf("got %q", got)
	}
}

func TestWritingTheSettingsBackDoesNotCarryTheKey(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"service":{"key":"sk-secret"}}}}`))
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

// A section named without a model is a section naming nothing.
func TestAProofreaderWithoutAModelIsNoProofreader(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{"use":"service"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Indexing.Proofreading.Named() {
		t.Error("proofreads with a model nobody named")
	}
}

func TestANamedProofreaderKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"proofreading":{"use":"service","service":{"name":"google/gemini-2.5-flash"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	read := cfg.Indexing.Proofreading
	if !read.Named() || read.Service.Name != "google/gemini-2.5-flash" {
		t.Errorf("got %+v", read)
	}
	if read.Service.BaseURL != proofreading.Defaults().Service.BaseURL {
		t.Errorf("base URL is %q", read.Service.BaseURL)
	}
	if read.Service.LettersApart != proofreading.Defaults().Service.LettersApart {
		t.Errorf("letters apart is %v", read.Service.LettersApart)
	}
}

func TestTheProofreadersKeyStaysOutOfWhatIsWrittenBack(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"proofreading":{"service":{"key":"sk-proof"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Proofreading.Service.Key(); got != "sk-proof" {
		t.Errorf("got %q", got)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-proof") {
		t.Errorf("the key is in %s", raw)
	}
	if strings.Contains(cfg.Indexing.Proofreading.Service.String(), "sk-proof") {
		t.Errorf("the key is in %s", cfg.Indexing.Proofreading.Service)
	}
}
