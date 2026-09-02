package settings

import (
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A model is nothing to the window unless it says where it is read from and
// what choosing it writes.
func TestEveryModelSaysWhereItIsReadAndWhatItWrites(t *testing.T) {
	for _, one := range append(Models(), Agents()...) {
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
	for _, one := range append(Models(), Agents()...) {
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
	held := models(t, EmbeddingModelAt)
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
	held := models(t, AgentModelAt)
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

// models are the models one setting can be set to, in the order they are
// offered.
func models(t *testing.T, setting []string) []port.Model {
	t.Helper()
	held := []port.Model{}
	for _, one := range append(Models(), Agents()...) {
		if at(one.NamedAt) == at(setting) {
			held = append(held, one)
		}
	}
	if len(held) == 0 {
		t.Fatalf("%s can be set to nothing", at(setting))
	}
	return held
}

// at is a path through the file, as one word.
func at(path []string) string { return strings.Join(path, ".") }
