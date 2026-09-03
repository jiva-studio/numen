package settings

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A model is nothing to the window unless it says where it is read from and
// what choosing it writes.
func TestEveryModelSaysWhereItIsReadAndWhatItWrites(t *testing.T) {
	for _, one := range append(Models(Defaults()), Agents()...) {
		if len(one.NamedAt) == 0 {
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
	for _, one := range append(Models(Defaults()), Agents()...) {
		if one.ByDefault {
			byDefault[at(one.NamedAt)]++
		}
	}
	for _, setting := range []string{
		at(EmbeddingModelAt), at(RecognitionModelAt), at(AgentModelAt), "agent.use",
	} {
		if byDefault[setting] != 1 {
			t.Errorf("%s has %d models by default", setting, byDefault[setting])
		}
	}
}

// The model the vault is indexed by is the one this application is built
// around, and choosing it writes the station that runs it.
func TestTheIndexingModelIsWrittenWithItsStation(t *testing.T) {
	held := models(t, Defaults(), EmbeddingModelAt)
	if len(held) != 1 {
		t.Fatalf("the vault is indexed by %d models", len(held))
	}
	written := []string{}
	for _, one := range held[0].Writes {
		written = append(written, at(one.At))
	}
	for _, want := range []string{
		"indexing.embedding.model",
		"indexing.embedding.indexing.use",
		"indexing.embedding.indexing.local.name",
	} {
		if !slices.Contains(written, want) {
			t.Errorf("choosing it leaves %s alone", want)
		}
	}
}

// The agent answers with whatever the machine answers with until a person says
// otherwise, and the sizes and the names in full stand on shelves of their own.
func TestTheAgentAnswersWithWhateverTheMachineAnswersWith(t *testing.T) {
	held := models(t, Defaults(), AgentModelAt)
	if held[0].Name != "" || !held[0].ByDefault {
		t.Errorf("the agent opens on %+v", held[0])
	}
	shelves := map[string]bool{}
	for _, one := range held[1:] {
		shelves[one.Shelf] = true
		if one.Name == "" {
			t.Errorf("a model past the first is named nothing")
		}
	}
	if len(shelves) != 2 {
		t.Errorf("the models stand on %d shelves", len(shelves))
	}
}

// A model whose weights stand in the folder the settings name is present.
func TestAModelWhoseFilesAreHereIsPresent(t *testing.T) {
	held := alone(t)
	dir := t.TempDir()
	written(t, filepath.Join(dir, embed.ModelFile))
	written(t, filepath.Join(dir, embed.TokenizerFile))
	held.Indexing.Embedding.Indexing.Local.Dir = dir

	if got := models(t, held, EmbeddingModelAt)[0].Presence; got != port.Present {
		t.Errorf("the model in %s stands at %v", dir, got)
	}
}

// A repository fetched into the cache stands under the snapshot it was fetched
// at, and that is where the row reads it.
func TestAModelInTheCacheIsPresent(t *testing.T) {
	held := alone(t)
	held.Indexing.Embedding.Indexing.Local.Dir = ""
	snapshot := filepath.Join(
		os.Getenv("XDG_CACHE_HOME"), "huggingface", "hub",
		"models--intfloat--multilingual-e5-small", "snapshots", "0a1b2c3d",
		embed.ModelFolder,
	)
	if err := os.MkdirAll(snapshot, 0o700); err != nil {
		t.Fatal(err)
	}
	written(t, filepath.Join(snapshot, embed.ModelFile))

	if got := models(t, held, EmbeddingModelAt)[0].Presence; got != port.Present {
		t.Errorf("the model in the cache stands at %v", got)
	}
}

// A model this machine runs and has not fetched is a wait, and the row says so.
func TestAModelWhoseFilesAreNotHereIsNotFetched(t *testing.T) {
	held := alone(t)
	held.Indexing.Embedding.Indexing.Local.Dir = t.TempDir()

	if got := models(t, held, EmbeddingModelAt)[0].Presence; got != port.NotFetched {
		t.Errorf("a model nothing has fetched stands at %v", got)
	}
}

// A model reached over the network is fetched nowhere, and where files stand
// says nothing about it.
func TestAModelReachedOverTheNetworkHasNothingToFetch(t *testing.T) {
	held := alone(t)
	held.Indexing.Embedding.Model.Name = "text-embedding-3-small"
	held.Indexing.Embedding.Indexing.Use = embed.UseService

	got := models(t, held, EmbeddingModelAt)
	one := got[len(got)-1]
	if one.Name != "text-embedding-3-small" || one.Presence != port.NothingToFetch {
		t.Errorf("the model a service answers with stands as %+v", one)
	}
	if answers := models(t, held, AgentModelAt)[0]; answers.Presence != port.NothingToFetch {
		t.Errorf("the model the agent answers with stands at %v", answers.Presence)
	}
}

// A cache nothing has been fetched into is a model not fetched, and asking
// leaves the folder as absent as it was.
func TestACacheThatIsNotThereIsAModelNotFetched(t *testing.T) {
	held := alone(t)
	dir := filepath.Join(t.TempDir(), "cache")
	held.Indexing.Recognition.Dir = dir

	if got := models(t, held, RecognitionModelAt)[0].Presence; got != port.NotFetched {
		t.Errorf("a model under a cache that is not there stands at %v", got)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("asking made %s", dir)
	}
}

// The model the settings name is answered for whether it is one of those
// offered or not.
func TestTheModelTheSettingsNameIsOfferedToo(t *testing.T) {
	held := alone(t)
	dir := t.TempDir()
	held.Indexing.Recognition.Dir = dir
	held.Indexing.Recognition.Recognise.Name = "https://example.invalid/reads-tamil.onnx"
	written(t, filepath.Join(dir, "reads-tamil.onnx"))

	got := models(t, held, RecognitionModelAt)
	one := got[len(got)-1]
	if one.Name != held.Indexing.Recognition.Recognise.Name {
		t.Fatalf("the model the settings name is offered as %+v", one)
	}
	if one.Presence != port.Present {
		t.Errorf("the file in %s stands at %v", dir, one.Presence)
	}
}

// alone is the defaults with the caches of this machine out of reach, so what a
// test reads is what the test wrote.
func alone(t *testing.T) Config {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	held := Defaults()
	held.Indexing.Recognition.Dir = t.TempDir()
	held.Indexing.Embedding.Indexing.Local.Dir = t.TempDir()
	return held
}

// written is an empty file where a fetch would have put one.
func written(t *testing.T, at string) {
	t.Helper()
	if err := os.WriteFile(at, nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

// models are the models one setting can be set to, in the order they are
// offered.
func models(t *testing.T, held Config, setting []string) []port.Model {
	t.Helper()
	found := []port.Model{}
	for _, one := range append(Models(held), Agents()...) {
		if at(one.NamedAt) == at(setting) {
			found = append(found, one)
		}
	}
	if len(found) == 0 {
		t.Fatalf("%s can be set to nothing", at(setting))
	}
	return found
}

// at is a path through the file, as one word.
func at(path []string) string { return strings.Join(path, ".") }
