package cards_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

const mammals = "decks/Mammals.md"

type editing struct {
	vs            vaulted
	read          cards.Read
	addCard       cards.AddCard
	editCard      cards.EditCard
	removeValue   cards.RemoveCardValue
	removeCard    cards.RemoveCard
	addSection    cards.AddSection
	renameSection cards.RenameSection
	removeSection cards.RemoveSection
	page          cards.GetCardPage
}

func newEditing(t *testing.T) editing {
	t.Helper()
	vs := newVaults(t)
	read := cards.NewRead(vs.ports.Readers, vs.queries)
	write := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, time.Now)
	return editing{vs: vs, read: read, addCard: cards.NewAddCard(read, write), editCard: cards.NewEditCard(read, write),
		removeValue: cards.NewRemoveCardValue(read, write), removeCard: cards.NewRemoveCard(read, write),
		addSection: cards.NewAddSection(read, write), renameSection: cards.NewRenameSection(read, write),
		removeSection: cards.NewRemoveSection(read, write), page: cards.NewGetCardPage(read)}
}

func (e editing) version(t *testing.T) cards.DeckVersion {
	t.Helper()
	got, err := e.read.Deck(t.Context(), e.vs.first, mammals)
	if err != nil {
		t.Fatal(err)
	}
	return cards.DeckVersion{Path: mammals, Fingerprint: got.Fingerprint}
}

func TestReadingCardsGivesARunOrOneCardWithTheFieldsAskedFor(t *testing.T) {
	e := newEditing(t)

	whole, err := e.page.Execute(t.Context(), e.vs.first, cards.Query{Path: mammals, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if whole.Total != 2 || len(whole.Cards) != 2 || len(whole.Sections) != 1 || whole.Sections[0] != "The ones with fur" {
		t.Fatalf("the deck read as %+v", whole)
	}
	if whole.Fingerprint != e.version(t).Fingerprint {
		t.Error("the fingerprint is not the one the deck stands at")
	}

	run, err := e.page.Execute(t.Context(), e.vs.first, cards.Query{Path: mammals, From: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Cards) != 1 || string(run.Cards[0].Mark) != "zpqrstvwxy" || run.Total != 2 {
		t.Errorf("a run of one from the second read as %+v", run)
	}

	one, err := e.page.Execute(t.Context(), e.vs.first, cards.Query{
		Path: mammals, Mark: "k7m2xq9fzp", From: 1, Fields: []string{"Name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Cards) != 1 || len(one.Cards[0].Values) != 1 || one.Cards[0].Values[0].Field != "Name" {
		t.Errorf("one card with one field read as %+v", one.Cards)
	}

	if _, err := e.page.Execute(t.Context(), e.vs.first, cards.Query{
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
		_, err := e.page.Execute(t.Context(), e.vs.first, cards.Query{Path: path, Limit: 50})
		if err == nil || err.Error() != path+": "+why {
			t.Errorf("%s was refused as %v, want %q", path, err, why)
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
		t.Errorf("the fingerprint handed back stands at %d bytes, and the deck is %d", added.Fingerprint.Size, len(body))
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
		cards.DeckVersion{Path: mammals, Fingerprint: domain.Fingerprint{Size: 1}}, "New")
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

func TestRemovingAValueAndACardTakesThemOutOfTheDeck(t *testing.T) {
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
