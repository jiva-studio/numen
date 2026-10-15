package container

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	adapteragent "github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Where each setting that names a model stands in the file: the section it
// belongs to, and the place the adapter that owns it reads it at.
var (
	embeddingModelAt = slices.Concat([]string{indexingAt, embeddingAt}, embed.ModelAt)
	agentModelAt     = slices.Concat([]string{agentAt}, adapteragent.ModelAt)
	agentUseAt       = slices.Concat([]string{agentAt}, adapteragent.UseAt)
)

// A section stands under the key the document spells on the field that holds
// it, and what a model row writes is put under that same word. A key said twice
// is a row writing a setting nothing reads.
func TestEverySectionKeyIsTheOneTheDocumentSpells(t *testing.T) {
	for _, one := range []struct {
		in    any
		field string
		at    string
	}{
		{Settings{}, "Indexing", indexingAt},
		{Settings{}, "Agent", agentAt},
		{Settings{}, "Importing", importingAt},
		{Indexing{}, "Embedding", embeddingAt},
		{Indexing{}, "Recognition", recognitionAt},
	} {
		field, there := reflect.TypeOf(one.in).FieldByName(one.field)
		if !there {
			t.Errorf("%T holds no %s", one.in, one.field)
			continue
		}
		if key := field.Tag.Get("json"); key != one.at {
			t.Errorf("%T.%s stands at %q and is read under %q", one.in, one.field, key, one.at)
		}
	}
}

// A model is nothing to the window unless it says where it is read from and
// what choosing it writes.
func TestEveryModelSaysWhereItIsReadAndWhatItWrites(t *testing.T) {
	for _, one := range getModels(DefaultSettings()) {
		if len(one.Path) == 0 {
			t.Errorf("%q is read from nowhere", one.Title)
		}
		if one.Title == "" {
			t.Errorf("the model named %q is drawn as nothing", one.Name)
		}
		if len(one.Writes) == 0 {
			t.Errorf("choosing %q writes nothing", one.Title)
		}
	}
}

// One setting, one model an installation nobody has configured runs on.
func TestOneModelToASettingIsTheDefault(t *testing.T) {
	byDefault := map[string]int{}
	for _, one := range getModels(DefaultSettings()) {
		if one.IsDefault {
			byDefault[formatPath(one.Path)]++
		}
	}
	for _, setting := range []string{
		formatPath(embeddingModelAt),
		formatPath(agentModelAt), formatPath(agentUseAt),
	} {
		if byDefault[setting] != 1 {
			t.Errorf("%s has %d models by default", setting, byDefault[setting])
		}
	}
}

// The model the vault is indexed by is the one this application is built
// around, and choosing it writes the provider that runs it.
func TestTheIndexingModelIsWrittenWithItsProvider(t *testing.T) {
	held := modelsAt(t, DefaultSettings(), embeddingModelAt)
	if len(held) < 1 {
		t.Fatalf("the vault is indexed by %d models", len(held))
	}
	written := []string{}
	for _, one := range held[0].Writes {
		written = append(written, formatPath(one.Path))
	}
	for _, want := range []string{
		"indexing.embedding.model.name",
		"indexing.embedding.indexing.use",
		"indexing.embedding.indexing.local.name",
	} {
		if !slices.Contains(written, want) {
			t.Errorf("choosing it leaves %s alone", want)
		}
	}
}

// An agent that reports no models offers the model that answers by default,
// alone, until a person names one.
func TestAnAgentThatReportsNoModelsOffersTheDefaultAlone(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	adapteragent.RefreshModels()
	held := modelsAt(t, DefaultSettings(), agentModelAt)
	if len(held) != 1 || held[0].Name != "" || !held[0].IsDefault {
		t.Errorf("the agent opens on %+v", held)
	}
}

// A model whose weights stand in the folder the settings name is present.
func TestAModelWhoseFilesAreHereIsPresent(t *testing.T) {
	held := newIsolatedSettings(t)
	dir := t.TempDir()
	writeEmptyFile(t, filepath.Join(dir, embed.ModelFile))
	writeEmptyFile(t, filepath.Join(dir, embed.TokenizerFile))
	held = setModelDir(t, held, dir)

	if got := modelsAt(t, held, embeddingModelAt)[0].Presence; got != port.Present {
		t.Errorf("the model in %s stands at %v", dir, got)
	}
}

// A repository fetched into the cache stands under the snapshot it was fetched
// at, and that is where the row reads it.
func TestAModelInTheCacheIsPresent(t *testing.T) {
	held := newIsolatedSettings(t)
	held = setModelDir(t, held, "")
	snapshot := filepath.Join(
		os.Getenv("XDG_CACHE_HOME"), "huggingface", "hub",
		"models--intfloat--multilingual-e5-small", "snapshots", "0a1b2c3d",
		embed.ModelFolder,
	)
	if err := os.MkdirAll(snapshot, 0o700); err != nil {
		t.Fatal(err)
	}
	writeEmptyFile(t, filepath.Join(snapshot, embed.ModelFile))

	if got := modelsAt(t, held, embeddingModelAt)[0].Presence; got != port.Present {
		t.Errorf("the model in the cache stands at %v", got)
	}
}

// A model this machine runs and has not fetched is a wait, and the row says so.
func TestAModelWhoseFilesAreNotHereIsNotFetched(t *testing.T) {
	held := setModelDir(t, newIsolatedSettings(t), t.TempDir())

	if got := modelsAt(t, held, embeddingModelAt)[0].Presence; got != port.NotFetched {
		t.Errorf("a model nothing has fetched stands at %v", got)
	}
}

// A model reached over the network is fetched nowhere, and where files stand
// says nothing about it.
func TestAModelReachedOverTheNetworkHasNothingToFetch(t *testing.T) {
	held := newIsolatedSettings(t)
	held.Indexing.Embedding.Model.Name = "text-embedding-3-small"
	held.Indexing.Embedding.Indexing.Use = embed.UseService

	got := modelsAt(t, held, embeddingModelAt)
	one := got[len(got)-1]
	if one.Name != "text-embedding-3-small" || one.Presence != port.NothingToFetch {
		t.Errorf("the model a service answers with stands as %+v", one)
	}
	if answers := modelsAt(t, held, agentModelAt)[0]; answers.Presence != port.NothingToFetch {
		t.Errorf("the model the agent answers with stands at %v", answers.Presence)
	}
}

// A provider reaching a service fetches nothing, whichever model a row names.
// The files of a model this machine once ran stay in the cache, and they say
// nothing about a vault indexed over the network.
func TestEveryRowOfAProviderReachingAServiceHasNothingToFetch(t *testing.T) {
	held := newIsolatedSettings(t)
	// The model the preset offers, fetched, and left where a fetch put it.
	dir := getModelDir(t, held)
	held.Indexing.Embedding.Indexing.Use = embed.UseService
	held.Indexing.Embedding.Model.Name = "baai/bge-m3"

	writeEmptyFile(t, filepath.Join(dir, embed.ModelFile))
	writeEmptyFile(t, filepath.Join(dir, embed.TokenizerFile))

	for _, one := range modelsAt(t, held, embeddingModelAt) {
		if one.Presence != port.NothingToFetch {
			t.Errorf("%q stands at %v", one.Name, one.Presence)
		}
	}
}

// Which of the two a provider is is `use`. A provider running a model here says
// where its files are, whatever the model is called.
func TestAProviderRunningAModelHereSaysWhereItsFilesAre(t *testing.T) {
	held := newIsolatedSettings(t)
	held.Indexing.Embedding.Indexing.Use = embed.UseLocal
	held.Indexing.Embedding.Model.Name = "somewhere/of-my-own"

	got := modelsAt(t, held, embeddingModelAt)
	if got[len(got)-1].Presence != port.NotFetched {
		t.Errorf("a model nothing has fetched stands at %v", got[len(got)-1].Presence)
	}
}

// newIsolatedSettings is the defaults with the caches of this machine out of reach, so what a
// test reads is what the test wrote.
func newIsolatedSettings(t *testing.T) Settings {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	held := DefaultSettings()
	return setModelDir(t, held, t.TempDir())
}

// setModelDir is the settings with the model this machine runs read out of dir.
func setModelDir(t *testing.T, held Settings, dir string) Settings {
	t.Helper()
	local, ok := held.Indexing.Embedding.Indexing.Local()
	if !ok {
		t.Fatal("these settings run no model on this machine")
	}
	local.Dir = dir
	held.Indexing.Embedding.Indexing = held.Indexing.Embedding.Indexing.SetLocal(local)
	return held
}

// getModelDir is the folder the settings read the model this machine runs out of.
func getModelDir(t *testing.T, held Settings) string {
	t.Helper()
	local, ok := held.Indexing.Embedding.Indexing.Local()
	if !ok {
		t.Fatal("these settings run no model on this machine")
	}
	return local.Dir
}

// writeEmptyFile is an empty file where a fetch would have put one.
func writeEmptyFile(t *testing.T, at string) {
	t.Helper()
	if err := os.WriteFile(at, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

// modelsAt are the models one setting can be set to, in the order they are
// offered.
func modelsAt(t *testing.T, held Settings, setting []string) []port.Model {
	t.Helper()
	found := []port.Model{}
	for _, one := range getModels(held) {
		if formatPath(one.Path) == formatPath(setting) {
			found = append(found, one)
		}
	}
	if len(found) == 0 {
		t.Fatalf("%s can be set to nothing", formatPath(setting))
	}
	return found
}

// formatPath is a path through the file, as one word.
func formatPath(path []string) string { return strings.Join(path, ".") }
