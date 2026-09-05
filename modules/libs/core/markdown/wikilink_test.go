package markdown_test

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// The brackets are written once and read twice: here, and in the window, where
// the editor underlines them and the renderer turns them into links. Both read
// this one corpus, and neither owns it.
const wikilinkCorpus = "../../protocol/testdata/wikilinks.json"

type wikilinkCase struct {
	Text  string   `json:"text"`
	Found []string `json:"found"`
}

func TestTheWikilinksInALineAreTheOnesTheSchemaSays(t *testing.T) {
	raw, err := os.ReadFile(wikilinkCorpus)
	if err != nil {
		t.Fatal(err)
	}
	var cases []wikilinkCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("the corpus is empty")
	}

	for _, one := range cases {
		found := []string{}
		for _, w := range markdown.WikilinksIn(one.Text) {
			found = append(found, w.Target.String())
		}
		if !slices.Equal(found, one.Found) {
			t.Errorf("%q holds %v, want %v", one.Text, found, one.Found)
		}
	}
}

func TestAWikilinkSaysWhereItStandsAndWhatIsInsideIt(t *testing.T) {
	found := markdown.WikilinksIn("Under [[Entropy|the other way]] it sits.")
	if len(found) != 1 {
		t.Fatalf("got %d wikilinks", len(found))
	}
	if found[0].At != 6 || found[0].To != 31 {
		t.Errorf("stands at %d..%d", found[0].At, found[0].To)
	}
	if found[0].Inside != "Entropy|the other way" {
		t.Errorf("holds %q", found[0].Inside)
	}
}
