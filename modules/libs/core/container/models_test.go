package container

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	adapteragent "github.com/jiva-studio/numen/modules/libs/core/adapter/agent"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Where each setting that names a model stands in the file: the section it
// belongs to, and the place the adapter that owns it reads it at.
var (
	agentModelAt = slices.Concat([]string{agentAt}, adapteragent.ModelAt)
	agentUseAt   = slices.Concat([]string{agentAt}, adapteragent.UseAt)
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
		formatPath(agentModelAt), formatPath(agentUseAt),
	} {
		if byDefault[setting] != 1 {
			t.Errorf("%s has %d models by default", setting, byDefault[setting])
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
