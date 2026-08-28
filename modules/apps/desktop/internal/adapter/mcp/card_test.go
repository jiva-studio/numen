package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// The operations an agent works a deck through: the cards of a file are read as
// cards, one is added, changed or taken out, and the file is written back with
// the wikilink under each heading and every card nobody touched as the bytes it
// was. These are what those operations do.

const stencil = "---\ntype: stencil\nfields:\n  - Name\n  - Height\n  - Life span\n---\n\n" +
	"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"

const deck = "---\ntype: deck\n---\n\n" +
	"## Llama\n\n[[Animal]]\n\n### Height\n\nabout 45\"\n\n" +
	"## Alpaca\n\nsomebody's prose\n\n### Height\n\nabout 36\"\n"

// vault is the two notes a flashcard is made of, and an ordinary note beside
// them.
func vault() map[string]string {
	return map[string]string{
		"Animal.md":  stencil,
		"Animals.md": deck,
		"Entropy.md": "# Entropy\n",
	}
}

// cards is what card_read answered.
type cards struct {
	Cards []struct {
		Name    string `json:"name"`
		Stencil string `json:"stencil"`
		Values  []struct {
			Field string `json:"field"`
			Text  string `json:"text"`
		} `json:"values"`
	} `json:"cards"`
	Held   int `json:"held"`
	Faults []struct {
		Card int    `json:"card"`
		Why  string `json:"why"`
	} `json:"faults"`
	Fingerprint string `json:"fingerprint"`
}

func dealt(t *testing.T, s *sdk.ClientSession, args map[string]any) cards {
	t.Helper()
	return call[cards](t, s, "card_read", args)
}

// held is what the vault holds at a path.
func held(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// TestADeckIsReadAsCardsAndNotAsMarkdown. What comes back is the fields a
// person filled in, named, so nothing has to parse a heading to find them.
func TestADeckIsReadAsCardsAndNotAsMarkdown(t *testing.T) {
	session, _ := connected(t, vault())

	read := dealt(t, session, map[string]any{"path": "Animals.md"})
	if read.Held != 2 {
		t.Fatalf("the deck came back holding %d cards", read.Held)
	}
	if read.Cards[0].Name != "Llama" || read.Cards[0].Stencil != "Animal" {
		t.Errorf("the first card came back as %+v", read.Cards[0])
	}
	if len(read.Cards[0].Values) != 1 {
		t.Fatalf("the first card holds %+v", read.Cards[0].Values)
	}
	if got := read.Cards[0].Values[0]; got.Field != "Height" || got.Text != "about 45\"" {
		t.Errorf("the first card holds %+v", got)
	}
	if read.Fingerprint == "" {
		t.Error("a deck was read with nothing to present at the next write of it")
	}
}

// TestAFaultStandsAgainstTheCardItIsAbout. A card is addressed by where it
// stands, so a fault says which card it is against.
func TestAFaultStandsAgainstTheCardItIsAbout(t *testing.T) {
	session, _ := connected(t, vault())

	read := dealt(t, session, map[string]any{"path": "Animals.md"})
	if len(read.Faults) != 1 {
		t.Fatalf("the deck came back with %d faults: %+v", len(read.Faults), read.Faults)
	}
	at := read.Faults[0].Card
	if at < 0 || at >= len(read.Cards) {
		t.Fatalf("the fault stands against card %d, and the deck answered with %d", at, len(read.Cards))
	}
	if name := read.Cards[at].Name; name != "Alpaca" {
		t.Errorf("the card with no wikilink under it is %q, and the fault stands against %q",
			"Alpaca", name)
	}
}

// TestADeckAnswersARunOfItsCardsAndSaysHowManyItHolds. A limit that truncates
// in silence reads as "that is all there is".
func TestADeckAnswersARunOfItsCardsAndSaysHowManyItHolds(t *testing.T) {
	session, _ := connected(t, vault())

	read := dealt(t, session, map[string]any{"path": "Animals.md", "limit": 1})
	if len(read.Cards) != 1 {
		t.Fatalf("a call asking for one card came back with %d", len(read.Cards))
	}
	if read.Held != 2 {
		t.Errorf("the deck holds two cards and the answer says %d", read.Held)
	}
	if read.Cards[0].Name != "Llama" {
		t.Errorf("the run began at %q", read.Cards[0].Name)
	}

	on := dealt(t, session, map[string]any{"path": "Animals.md", "from": 1, "limit": 1})
	if len(on.Cards) != 1 || on.Cards[0].Name != "Alpaca" {
		t.Errorf("the next run came back as %+v", on.Cards)
	}

	if said := failing(t, session, "card_read", map[string]any{
		"path": "Animals.md", "limit": 500,
	}); !strings.Contains(said, "at most") {
		t.Errorf("a call over the ceiling was answered %q", said)
	}
}

// TestACardIsAddedWithTheWikilinkThatNamesItsStencil. A card is cut by the
// stencil that link names, and one written under no link is cut by none.
func TestACardIsAddedWithTheWikilinkThatNamesItsStencil(t *testing.T) {
	session, v := connected(t, vault())

	call[map[string]any](t, session, "card_add", map[string]any{
		"path": "Animals.md", "name": "Vicuña", "stencil": "Animal",
		"values": []map[string]string{{"field": "Height", "text": "about 34\""}},
	})

	written := held(t, v, "Animals.md")
	if !strings.Contains(written, "## Vicuña\n\n[[Animal]]\n\n### Height\n\nabout 34\"\n") {
		t.Errorf("the card was written as %q", written)
	}
	read := dealt(t, session, map[string]any{"path": "Animals.md"})
	if read.Held != 3 || read.Cards[2].Stencil != "Animal" {
		t.Errorf("the deck now holds %+v", read.Cards)
	}
}

// TestACardIsEditedAndTheCardsBesideItAreLeftAlone. The prose somebody wrote
// under another card's wikilink is theirs, and no write of ours moves it.
func TestACardIsEditedAndTheCardsBesideItAreLeftAlone(t *testing.T) {
	session, v := connected(t, vault())

	call[map[string]any](t, session, "card_edit", map[string]any{
		"path": "Animals.md", "card": "Llama",
		"values": []map[string]string{
			{"field": "Height", "text": "about 46\""},
			{"field": "Life span", "text": "about 20 years"},
		},
	})

	written := held(t, v, "Animals.md")
	if !strings.Contains(written, "### Height\n\nabout 46\"\n") {
		t.Errorf("the value was written as %q", written)
	}
	if !strings.Contains(written, "### Life span\n\nabout 20 years\n") {
		t.Errorf("a field the card did not carry was written as %q", written)
	}
	if !strings.Contains(written, "## Alpaca\n\nsomebody's prose\n") {
		t.Errorf("the card beside it came back as %q", written)
	}
}

// TestACardIsRemovedAndNothingElseIs.
func TestACardIsRemovedAndNothingElseIs(t *testing.T) {
	session, v := connected(t, vault())

	call[map[string]any](t, session, "card_remove", map[string]any{
		"path": "Animals.md", "card": "Llama",
	})

	written := held(t, v, "Animals.md")
	if strings.Contains(written, "## Llama") {
		t.Errorf("the card is still in the file: %q", written)
	}
	if !strings.Contains(written, "## Alpaca\n\nsomebody's prose\n\n### Height\n\nabout 36\"\n") {
		t.Errorf("the card beside it came back as %q", written)
	}
	if said := failing(t, session, "card_remove", map[string]any{
		"path": "Animals.md", "card": "Llama",
	}); !strings.Contains(said, "no card") {
		t.Errorf("removing a card twice was answered %q", said)
	}
}

// TestAWriteIsRefusedOverAnEditNobodySaw. Somebody editing their own deck
// outranks an agent that read it, thought about it and arrived late.
func TestAWriteIsRefusedOverAnEditNobodySaw(t *testing.T) {
	session, v := connected(t, vault())

	read := dealt(t, session, map[string]any{"path": "Animals.md"})
	theirs := deck + "\n## Guanaco\n\n[[Animal]]\n\n### Height\n\nabout 43\"\n"
	if err := os.WriteFile(filepath.Join(v.Path, "Animals.md"), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	said := failing(t, session, "card_add", map[string]any{
		"path": "Animals.md", "name": "Vicuña", "stencil": "Animal",
		"values":      []map[string]string{{"field": "Height", "text": "about 34\""}},
		"fingerprint": read.Fingerprint,
	})
	if !strings.Contains(said, "changed") {
		t.Errorf("a write over an edit nobody saw was answered %q", said)
	}
	if written := held(t, v, "Animals.md"); written != theirs {
		t.Errorf("the deck on disk is now %q", written)
	}
}

// TestTheStencilsOfAVaultAreListedWithWhatTheyAskFor. A card names its stencil
// and carries the fields that stencil declares, so both travel together.
func TestTheStencilsOfAVaultAreListedWithWhatTheyAskFor(t *testing.T) {
	session, _ := connected(t, vault())

	listed := call[struct {
		Stencils []struct {
			Path   string   `json:"path"`
			Title  string   `json:"title"`
			Fields []string `json:"fields"`
		} `json:"stencils"`
		Held int `json:"held"`
	}](t, session, "card_stencils", map[string]any{})

	if listed.Held != 1 || len(listed.Stencils) != 1 {
		t.Fatalf("the vault holds one stencil and the answer is %+v", listed)
	}
	if listed.Stencils[0].Path != "Animal.md" {
		t.Errorf("the stencil came back as %+v", listed.Stencils[0])
	}
	if got := listed.Stencils[0].Fields; len(got) != 3 || got[0] != "Name" || got[2] != "Life span" {
		t.Errorf("the stencil asks for %v", got)
	}
}

// TestAStencilIsMadeAsAStencilAndNotAsANote. One key says what a note is, and
// the file carries it from the moment it exists.
func TestAStencilIsMadeAsAStencilAndNotAsANote(t *testing.T) {
	session, v := connected(t, vault())

	made := call[struct {
		Path string `json:"Path"`
	}](t, session, "card_stencil_create", map[string]any{
		"title":  "Term",
		"fields": []string{"Word", "Meaning"},
		"faces": []map[string]string{
			{"name": "Recall", "front": "{{Word}}", "back": "{{Meaning}}"},
		},
	})
	if made.Path != "Term.md" {
		t.Fatalf("the stencil was filed at %q", made.Path)
	}

	written := held(t, v, made.Path)
	if !strings.Contains(written, "type: stencil") {
		t.Errorf("the file says it is %q", written)
	}
	if !strings.Contains(written, "## Recall\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Meaning}}\n") {
		t.Errorf("the face was written as %q", written)
	}

	listed := call[struct {
		Held int `json:"held"`
	}](t, session, "card_stencils", map[string]any{})
	if listed.Held != 2 {
		t.Errorf("a stencil made through the tools is not on the list: %d", listed.Held)
	}
}

// TestRenamingAFieldReachesTheCardsCutByThatStencil. A field's name is written
// twice over, and renaming it in one place alone leaves values under a heading
// nothing declares.
func TestRenamingAFieldReachesTheCardsCutByThatStencil(t *testing.T) {
	session, v := connected(t, vault())

	renamed := call[struct {
		Decks []string `json:"decks"`
		Cards int      `json:"cards"`
	}](t, session, "card_rename_field", map[string]any{
		"path": "Animal.md", "from": "Height", "to": "Shoulder height",
	})
	if renamed.Cards != 1 || len(renamed.Decks) != 1 {
		t.Fatalf("the rename reached %+v", renamed)
	}

	if written := held(t, v, "Animal.md"); !strings.Contains(written, "- Shoulder height") {
		t.Errorf("the stencil declares %q", written)
	}
	written := held(t, v, "Animals.md")
	if !strings.Contains(written, "## Llama\n\n[[Animal]]\n\n### Shoulder height\n\nabout 45\"\n") {
		t.Errorf("the card cut by that stencil came back as %q", written)
	}
	// The card under no wikilink is cut by no stencil, so the rename is none of
	// its business.
	if !strings.Contains(written, "## Alpaca\n\nsomebody's prose\n\n### Height\n") {
		t.Errorf("a card of no stencil was rewritten: %q", written)
	}
}

// A deck is made through the tools an agent has, and filled through them.
//
// The window and the tools are served one set, so a deck a person can make is
// a deck an agent can make.
func TestADeckIsMadeThroughTheTools(t *testing.T) {
	session, v := connected(t, vault())

	made := call[struct {
		Path string `json:"Path"`
	}](t, session, "card_deck_create", map[string]any{
		"title": "Terms", "folder": "decks",
	})
	if made.Path != "decks/Terms.md" {
		t.Fatalf("the deck was filed at %q", made.Path)
	}
	if written := held(t, v, made.Path); !strings.Contains(written, "type: deck") {
		t.Errorf("the file says it is %q", written)
	}

	call[map[string]any](t, session, "card_add", map[string]any{
		"path": made.Path, "name": "Vicuña", "stencil": "Animal",
		"values": []map[string]string{{"field": "Height", "text": "about 34\""}},
	})
	if read := dealt(t, session, map[string]any{"path": made.Path}); read.Held != 1 {
		t.Errorf("the deck made through the tools holds %d cards", read.Held)
	}
}

// titled is a stencil whose title is not what its file is called.
const titled = "---\ntype: stencil\ntitle: Beast\nfields:\n  - Name\n  - Height\n---\n\n" +
	"## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n"

// The name a stencil is listed under is the name a card's wikilink resolves by.
//
// A link is resolved by path and by filename, so a name taken from anywhere
// else lands on nothing and every card written under it is cut by no stencil.
func TestAStencilIsListedByTheNameACardsWikilinkReaches(t *testing.T) {
	session, _ := connected(t, map[string]string{
		"Animal.md": titled,
		"Deck.md":   "---\ntype: deck\n---\n",
	})

	listed := call[struct {
		Stencils []struct {
			Path string `json:"path"`
			Name string `json:"name"`
		} `json:"stencils"`
	}](t, session, "card_stencils", map[string]any{})
	if len(listed.Stencils) != 1 {
		t.Fatalf("the vault holds one stencil and the answer is %+v", listed)
	}
	name := listed.Stencils[0].Name
	if name != "Animal" {
		t.Errorf("the stencil at %s is named %q", listed.Stencils[0].Path, name)
	}

	call[map[string]any](t, session, "card_add", map[string]any{
		"path": "Deck.md", "name": "Llama", "stencil": name,
		"values": []map[string]string{{"field": "Height", "text": "about 45\""}},
	})
	// A rename reaches the cards that stencil cuts, and a card whose wikilink
	// lands nowhere is cut by none.
	renamed := call[struct {
		Cards int `json:"cards"`
	}](t, session, "card_rename_field", map[string]any{
		"path": "Animal.md", "from": "Height", "to": "Shoulder height",
	})
	if renamed.Cards != 1 {
		t.Errorf("the card written under %q is cut by no stencil: the rename reached %d cards",
			name, renamed.Cards)
	}
}

// The list of stencils says what it answers with. A ceiling that is not said is
// a short list read as the whole vault.
func TestTheListOfStencilsSaysItsCeiling(t *testing.T) {
	session, _ := connected(t, vault())

	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var said string
	for _, tool := range listed.Tools {
		if tool.Name != "card_stencils" {
			continue
		}
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		said = tool.Description + string(schema)
	}
	if said == "" {
		t.Fatal("card_stencils is not among the tools")
	}
	if strings.Contains(said, "all of them") || strings.Contains(said, "Every stencil in the vault") {
		t.Errorf("the list answers with at most fifty and says it answers with every one:\n%s", said)
	}
	if !strings.Contains(said, "50") {
		t.Errorf("the ceiling the list answers under is not said:\n%s", said)
	}
}

// A card is named by its heading and by nothing else, so writing the naming
// field into the card as well is a fault against the deck. These tools exist so
// that an agent cannot write one.
func TestTheFieldACardIsNamedByIsNotWrittenTwice(t *testing.T) {
	session, v := connected(t, vault())

	said := failing(t, session, "card_edit", map[string]any{
		"path": "Animals.md", "card": "Llama",
		"values": []map[string]string{{"field": "Name", "text": "Llama"}},
	})
	if !strings.Contains(said, "Name") {
		t.Errorf("writing the naming field again was answered %q", said)
	}
	if written := held(t, v, "Animals.md"); strings.Contains(written, "### Name") {
		t.Errorf("the card carries its name twice: %q", written)
	}
}
