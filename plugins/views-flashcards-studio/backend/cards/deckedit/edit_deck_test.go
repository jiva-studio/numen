package deckedit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards/deckedit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

const mammals = "decks/Mammals.md"

var animals = map[string]string{
	"Animal.md": "---\n" +
		"type: stencil\n" +
		"mine: keep me verbatim\n" +
		"fields:\n" +
		"  - Name\n" +
		"  - Height\n" +
		"  - Life span\n" +
		"---\n" +
		"\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
	"Term.md": "---\ntype: stencil\nfields:\n  - Word\n  - Height\n---\n" +
		"\n## Say it\n\n### Front\n\n{{Word}}\n\n### Back\n\n{{Height}}\n",
	"decks/Mammals.md": "---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: deck\nmine: keep me verbatim\n---\n" +
		"\nCards I am learning.\n" +
		"\n# The ones with fur\n" +
		"\nThe section says something of its own.\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n### Height\n\nabout 45\"\n" +
		"\n### Life span\n\nabout 20 years\n" +
		"\n## Gloss ^zpqrstvwxy\n\n[[Term]]\n\n### Height\n\nnot a length at all\n" +
		"\n### Word\n\nGloss\n",
	"decks/Birds.md": "---\ntype: deck\n---\n" +
		"\n## Wren ^3f4g5h6j7k\n\n[[Animal]]\n\n### Name\n\nWren\n\n### Height\n\nabout 4\"\n",
	"Weather.md": "# Weather\n\nNo card in here.\n",
}

type vaulted struct {
	first   domain.Vault
	hosts   map[domain.VaultID]*hosttest.Vault
	ports   hostports.Ports
	queries vaulttest.Queries
}

func newVaults(t *testing.T) vaulted {
	t.Helper()
	first := vaulttest.New(t, animals)
	return vaulted{
		first: first.Vault, ports: first.Ports, queries: first.Queries,
		hosts: map[domain.VaultID]*hosttest.Vault{first.Vault.ID: first.Host},
	}
}

func indexNothing(context.Context, domain.Vault, []string) error { return nil }

func read(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type editing struct {
	vs            vaulted
	read          cards.Read
	addCard       deckedit.AddCard
	editCard      deckedit.EditCard
	removeValue   deckedit.RemoveCardValue
	removeCard    deckedit.RemoveCard
	addSection    deckedit.AddSection
	renameSection deckedit.RenameSection
	removeSection deckedit.RemoveSection
	page          deckedit.GetCardPage
}

func newEditing(t *testing.T) editing {
	t.Helper()
	vs := newVaults(t)
	read := cards.NewRead(vs.ports.Readers, vs.queries)
	write := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, time.Now)
	return editing{
		vs: vs, read: read,
		addCard:       deckedit.NewAddCard(read, write),
		editCard:      deckedit.NewEditCard(read, write),
		removeValue:   deckedit.NewRemoveCardValue(read, write),
		removeCard:    deckedit.NewRemoveCard(read, write),
		addSection:    deckedit.NewAddSection(read, write),
		renameSection: deckedit.NewRenameSection(read, write),
		removeSection: deckedit.NewRemoveSection(read, write),
		page:          deckedit.NewGetCardPage(read),
	}
}

func (e editing) version(t *testing.T) deckedit.DeckVersion {
	t.Helper()
	got, err := e.read.Deck(t.Context(), e.vs.first, mammals)
	if err != nil {
		t.Fatal(err)
	}
	return deckedit.DeckVersion{Path: mammals, Fingerprint: got.Fingerprint}
}

func TestReadingCardsProducesARunOrOneCardWithTheFieldsAskedFor(t *testing.T) {
	e := newEditing(t)

	whole, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if whole.Total != 2 || len(whole.Cards) != 2 || len(whole.Sections) != 1 || whole.Sections[0] != "The ones with fur" {
		t.Fatalf("the deck read as %+v", whole)
	}
	if whole.Fingerprint != e.version(t).Fingerprint {
		t.Error("fingerprint mismatch for deck")
	}

	run, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, From: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Cards) != 1 || string(run.Cards[0].Mark) != "zpqrstvwxy" || run.Total != 2 {
		t.Errorf("a run of one from the second read as %+v", run)
	}

	one, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{
		Path: mammals, Mark: "k7m2xq9fzp", From: 1, Fields: []string{"Name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Cards) != 1 || len(one.Cards[0].Values) != 1 || one.Cards[0].Values[0].Field != "Name" {
		t.Errorf("one card with one field read as %+v", one.Cards)
	}

	if _, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{
		Path: mammals, Mark: "nothing",
	}); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("a mark no card carries read as %v", err)
	}
}

func TestReadingAFileThatIsNoDeckIsRefusedByPath(t *testing.T) {
	e := newEditing(t)

	for path, why := range map[string]string{
		"Weather.md": "this note is not a deck",
		"Animal.md":  "this note is not a deck",
		"gone.md":    "there is no note at this path",
	} {
		_, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: path, Limit: 50})
		if err == nil || err.Error() != path+": "+why {
			t.Errorf("%s failed as %v, want %q", path, err, why)
		}
	}
}

func TestAddingACardWritesItAtTheEndAndNamesItsMark(t *testing.T) {
	e := newEditing(t)

	added, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal",
		[]format.Value{{Field: "Name", Text: "Okapi"}, {Field: "Height", Text: "about 60\""}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if added.Cards != 3 || added.Mark == "" {
		t.Fatalf("the deck became %+v", added)
	}
	body := read(t, e.vs.first, mammals)
	if !strings.Contains(body, "Okapi") || !strings.Contains(body, added.Mark) {
		t.Errorf("the card is not in the deck under its mark:\n%s", body)
	}
	if added.Fingerprint.Size != int64(len(body)) {
		t.Errorf("returned fingerprint has size %d, want %d", added.Fingerprint.Size, len(body))
	}
}

func TestAddingACardAtASectionThatIsNotThereIsRefused(t *testing.T) {
	e := newEditing(t)
	before := read(t, e.vs.first, mammals)
	nine := 9

	_, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal",
		[]format.Value{{Field: "Name", Text: "Okapi"}}, &nine)
	if err == nil || !strings.Contains(err.Error(), "there is none standing at 9") {
		t.Errorf("a section the deck does not hold was written to: %v", err)
	}
	if read(t, e.vs.first, mammals) != before {
		t.Error("the deck was written anyway")
	}
}

func TestADeckEditOverAnEditNobodySawIsRefused(t *testing.T) {
	e := newEditing(t)
	before := read(t, e.vs.first, mammals)

	_, err := e.addSection.Execute(t.Context(), e.vs.first,
		deckedit.DeckVersion{Path: mammals, Fingerprint: domain.Fingerprint{Size: 1}}, "New")
	if !errors.Is(err, ports.ErrStale) {
		t.Errorf("a write over a deck that changed since was %v", err)
	}
	if read(t, e.vs.first, mammals) != before {
		t.Error("the deck was written anyway")
	}
}

func TestEditingACardReplacesAndAddsValues(t *testing.T) {
	e := newEditing(t)

	edited, err := e.editCard.Execute(t.Context(), e.vs.first, e.version(t), "k7m2xq9fzp", "", []format.Value{{Field: "Height", Text: "about 120 cm"}})
	if err != nil {
		t.Fatal(err)
	}
	if edited.Cards != 2 {
		t.Errorf("the deck holds %d cards", edited.Cards)
	}
	if body := read(t, e.vs.first, mammals); !strings.Contains(body, "about 120 cm") || strings.Contains(body, "about 45") {
		t.Errorf("the value was not replaced:\n%s", body)
	}

	if _, err := e.editCard.Execute(t.Context(), e.vs.first, e.version(t), "nothing", "", []format.Value{{Field: "Height", Text: "x"}}); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("a card that is not there was edited: %v", err)
	}
}

func TestRemovingAValueAndACardConsumesThemOutOfTheDeck(t *testing.T) {
	e := newEditing(t)

	if _, err := e.removeValue.Execute(t.Context(), e.vs.first, e.version(t), "k7m2xq9fzp", "Life span"); err != nil {
		t.Fatal(err)
	}
	if body := read(t, e.vs.first, mammals); strings.Contains(body, "about 20 years") {
		t.Errorf("the value stands:\n%s", body)
	}

	removed, err := e.removeCard.Execute(t.Context(), e.vs.first, e.version(t), "k7m2xq9fzp")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Cards != 1 {
		t.Errorf("the deck holds %d cards", removed.Cards)
	}
	if body := read(t, e.vs.first, mammals); strings.Contains(body, "k7m2xq9fzp") {
		t.Errorf("the card stands:\n%s", body)
	}
}

func TestSectionsAreAddedRenamedAndRemovedWithoutTouchingCards(t *testing.T) {
	e := newEditing(t)

	if _, err := e.addSection.Execute(t.Context(), e.vs.first, e.version(t), "The feathered"); err != nil {
		t.Fatal(err)
	}
	if body := read(t, e.vs.first, mammals); !strings.Contains(body, "# The feathered") {
		t.Errorf("the section was not written:\n%s", body)
	}
	if _, err := e.renameSection.Execute(t.Context(), e.vs.first, e.version(t), 0, "The furry"); err != nil {
		t.Fatal(err)
	}
	if body := read(t, e.vs.first, mammals); !strings.Contains(body, "# The furry") || strings.Contains(body, "# The ones with fur") {
		t.Errorf("the section was not renamed:\n%s", body)
	}
	removed, err := e.removeSection.Execute(t.Context(), e.vs.first, e.version(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Cards != 2 {
		t.Errorf("removing a section left %d cards", removed.Cards)
	}
	if body := read(t, e.vs.first, mammals); strings.Contains(body, "# The feathered") {
		t.Errorf("the section stands:\n%s", body)
	}
}
