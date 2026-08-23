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
	if cfg.Indexing.Embedding.Indexing.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Indexing.Use)
	}
	if cfg.Indexing.Embedding.Indexing.Local.Name == "" || cfg.Indexing.Embedding.Model.Dimensions == 0 {
		t.Errorf("no model to run: %+v", cfg.Indexing.Embedding)
	}
	// Zero is what says the desktop decides, so nothing may fill it in.
	if cfg.Appearance.Zoom != 0 {
		t.Errorf("zoom is %v", cfg.Appearance.Zoom)
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
	if !back.Indexing.Embedding.Indexing.Local.Download {
		t.Error("a machine with no model would fetch none")
	}
	// A key nobody set is not a field of the file.
	if strings.Contains(string(raw), `"key"`) {
		t.Errorf("the file offers a key:\n%s", raw)
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
	if cfg.Indexing.Embedding.Indexing.Local.Name != embed.Defaults().Indexing.Local.Name {
		t.Errorf("local model is %q", cfg.Indexing.Embedding.Indexing.Local.Name)
	}
	if cfg.Indexing.Embedding.Indexing.Use != embed.UseLocal {
		t.Errorf("uses %q", cfg.Indexing.Embedding.Indexing.Use)
	}
}

// A file naming one embedder writes the model among that embedder's fields.
func TestAFileNamingOneFieldKeepsTheDefaultsForTheRest(t *testing.T) {
	cfg, err := settings.At(write(t,
		`{"indexing":{"embedding":{"use":"service","service":{"name":"text-embedding-3-large","dimensions":3072}}}}`))
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
	if e.Indexing.Service.Name != "text-embedding-3-large" {
		t.Errorf("got %+v", e.Indexing.Service)
	}
	if e.Indexing.Service.BaseURL != embed.Defaults().Indexing.Service.BaseURL {
		t.Errorf("base URL is %q", e.Indexing.Service.BaseURL)
	}
	if e.Indexing.Local.Name != embed.Defaults().Indexing.Local.Name {
		t.Errorf("local model is %q", e.Indexing.Local.Name)
	}
	// A question is asked the way the vault was indexed.
	if got := e.Asking(); got.Use != embed.UseService {
		t.Errorf("questions are embedded by %+v", got)
	}
}

// The commonest flat file: the service named, and a key. What the model is
// comes from the placement the file names.
func TestAFlatFileNamingOnlyAKeyIsTheServicesModel(t *testing.T) {
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"use":"service","service":{"key":"sk-x"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	e := cfg.Indexing.Embedding
	if e.Model.Name != e.Indexing.Service.Name {
		t.Errorf("the model is %q and the service is %q", e.Model.Name, e.Indexing.Service.Name)
	}
	if e.Model.Dimensions != embed.ServedDimensions {
		t.Errorf("the model is %d wide", e.Model.Dimensions)
	}
	// A service says where it cuts a text off, or it says nothing.
	if e.Model.MaxTokens != 0 {
		t.Errorf("the model cuts at %d", e.Model.MaxTokens)
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
	if e.Indexing.Use != embed.UseService || e.Indexing.Service.Name != "baai/bge-m3" {
		t.Errorf("indexed by %+v", e.Indexing)
	}
	if got := e.Asking(); got.Use != embed.UseLocal || got.Local.Name != "BAAI/bge-m3" {
		t.Errorf("asked by %+v", got)
	}
	// The section nobody wrote about keeps its default.
	if e.Indexing.Service.BatchCharacters != embed.Defaults().Indexing.Service.BatchCharacters {
		t.Errorf("batch is %d", e.Indexing.Service.BatchCharacters)
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

	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"service":{"key":"from-the-file"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Indexing.Service.Key(); got != "from-the-file" {
		t.Errorf("got %q", got)
	}

	cfg, err = settings.At(write(t, `{"indexing":{"embedding":{"service":{}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Indexing.Service.Key(); got != "from-the-environment" {
		t.Errorf("got %q", got)
	}
}

func TestAnInstallationMayNameItsOwnEnvironmentVariable(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "from-openrouter")
	cfg, err := settings.At(write(t, `{"indexing":{"embedding":{"service":{"key_env":"OPENROUTER_API_KEY"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Indexing.Embedding.Indexing.Service.Key(); got != "from-openrouter" {
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
