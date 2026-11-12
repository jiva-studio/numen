package deckedit_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/hostports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/cards/deckedit"
)

const mammals = "decks/Mammals.md"

func getAnimals() map[string]string {
	return map[string]string{
		"Animal.md": "---\n" +
			"type: card-model\n" +
			"mine: keep me verbatim\n" +
			"fields:\n" +
			"  - Name\n" +
			"  - Height\n" +
			"  - Life span\n" +
			"---\n" +
			"\n## Recognize\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Height}}\n",
		"Term.md": "---\ntype: card-model\nfields:\n  - Word\n  - Height\n---\n" +
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
}

type vaulted struct {
	first   domain.Vault
	hosts   map[domain.VaultID]*hosttest.Vault
	ports   hostports.Ports
	queries vaulttest.Queries
}

func newVaults(t *testing.T) vaulted {
	t.Helper()
	first := vaulttest.New(t, getAnimals())
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

type editorFixture struct {
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

func newEditorFixture(t *testing.T) editorFixture {
	t.Helper()
	vs := newVaults(t)
	read := cards.NewRead(vs.ports.Readers, vs.queries)
	write := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, vaulttest.NewClock())
	return editorFixture{
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

func (e editorFixture) version(t *testing.T) deckedit.DeckVersion {
	t.Helper()
	got, err := e.read.Deck(t.Context(), e.vs.first, mammals)
	if err != nil {
		t.Fatal(err)
	}
	return deckedit.DeckVersion{Path: mammals, FileStat: got.FileStat}
}

func TestPageCards_ReturnsRunOrSingleCardWithRequestedFields(t *testing.T) {
	e := newEditorFixture(t)

	whole, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if whole.Total != 2 || len(whole.Cards) != 2 || len(whole.Sections) != 1 || whole.Sections[0] != "The ones with fur" {
		t.Fatalf("the deck read as %+v", whole)
	}
	if whole.FileStat != e.version(t).FileStat {
		t.Error("fingerprint mismatch for deck")
	}

	run, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, From: 1, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Cards) != 1 || string(run.Cards[0].CardID) != "zpqrstvwxy" || run.Total != 2 {
		t.Errorf("a run of one from the second read as %+v", run)
	}

	one, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{
		Path: mammals, CardID: "k7m2xq9fzp", From: 1, Fields: []string{"Name"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Cards) != 1 || len(one.Cards[0].Values) != 1 || one.Cards[0].Values[0].Field != "Name" {
		t.Errorf("one card with one field read as %+v", one.Cards)
	}

	if _, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{
		Path: mammals, CardID: "nothing",
	}); !errors.Is(err, format.ErrNoSuchCard) {
		t.Errorf("a card id no card carries read as %v", err)
	}
}

func TestReadingAFileThatIsNoDeckIsRefusedByPath(t *testing.T) {
	e := newEditorFixture(t)

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
	e := newEditorFixture(t)

	added, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal",
		[]format.Value{{Field: "Name", Text: "Okapi"}, {Field: "Height", Text: "about 60\""}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if added.Cards != 3 || added.CardID == "" {
		t.Fatalf("the deck became %+v", added)
	}
	body := read(t, e.vs.first, mammals)
	if !strings.Contains(body, "Okapi") || !strings.Contains(body, added.CardID) {
		t.Errorf("the card is not in the deck under its card id:\n%s", body)
	}
	if added.FileStat.Size != int64(len(body)) {
		t.Errorf("returned fingerprint has size %d, want %d", added.FileStat.Size, len(body))
	}
}

func TestAddingACardAtASectionThatIsNotThereIsRefused(t *testing.T) {
	e := newEditorFixture(t)
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

func TestADeckEditOverAnEditThatWasNeverReadIsRefused(t *testing.T) {
	e := newEditorFixture(t)
	before := read(t, e.vs.first, mammals)

	_, err := e.addSection.Execute(t.Context(), e.vs.first,
		deckedit.DeckVersion{Path: mammals, FileStat: domain.FileStat{Size: 1}}, "New")
	if !errors.Is(err, ports.ErrStale) {
		t.Errorf("a write over a deck that changed since was %v", err)
	}
	if read(t, e.vs.first, mammals) != before {
		t.Error("the deck was written anyway")
	}
}

func TestEditingACardReplacesAndAddsValues(t *testing.T) {
	e := newEditorFixture(t)

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
	e := newEditorFixture(t)

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
	e := newEditorFixture(t)

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

func TestDeckEditScenariosCoverEdgesAndErrors(t *testing.T) {
	e := newEditorFixture(t)

	// Query second card by CardID (pos == 1) and with empty Fields (len(fields) == 0).
	second, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{
		Path: mammals, CardID: "zpqrstvwxy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Cards) != 1 || string(second.Cards[0].CardID) != "zpqrstvwxy" || len(second.Cards[0].Values) != 2 {
		t.Fatalf("got %+v, want second card with 2 values", second.Cards)
	}

	// AddCard at section 0, negative section (-1), and exact out-of-bounds section (1).
	zero := 0
	addedInSec, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal",
		[]format.Value{{Field: "Name", Text: "Alpaca"}}, &zero)
	if err != nil {
		t.Fatal(err)
	}
	if addedInSec.Cards != 3 || addedInSec.CardID == "" {
		t.Fatalf("got %+v", addedInSec)
	}

	minusOne := -1
	if _, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal", nil, &minusOne); err == nil {
		t.Fatal("expected error for section -1")
	}
	oneSec := 1
	if _, err := e.addCard.Execute(t.Context(), e.vs.first, e.version(t), "Animal", nil, &oneSec); err == nil {
		t.Fatal("expected error for section == len(Sections)")
	}

	// EditCard with non-empty cardModel.
	if _, err := e.editCard.Execute(t.Context(), e.vs.first, e.version(t), "k7m2xq9fzp", "Term",
		[]format.Value{{Field: "Word", Text: "Llama"}}); err != nil {
		t.Fatal(err)
	}
	if body := read(t, e.vs.first, mammals); !strings.Contains(body, "[[Term]]") {
		t.Fatalf("expected card model Term in body:\n%s", body)
	}

	// Duplicate CardID in deck triggers ErrTwoCards at pos 0 and pos > 0.
	dupBody := "---\ntype: deck\n---\n\n## One ^k7m2xq9fzp\n\n[[Animal]]\n\n## Two ^k7m2xq9fzp\n\n[[Animal]]\n\n## Three ^zpqrstvwxy\n\n[[Animal]]\n\n## Four ^zpqrstvwxy\n\n[[Animal]]\n"
	if err := os.WriteFile(filepath.Join(e.vs.first.Path, filepath.FromSlash(mammals)), []byte(dupBody), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, CardID: "k7m2xq9fzp"}); !errors.Is(err, format.ErrTwoCards) {
		t.Fatalf("got %v, want ErrTwoCards for first card duplicate", err)
	}
	if _, err := e.page.Execute(t.Context(), e.vs.first, deckedit.Query{Path: mammals, CardID: "zpqrstvwxy"}); !errors.Is(err, format.ErrTwoCards) {
		t.Fatalf("got %v, want ErrTwoCards for later card duplicate", err)
	}

	// Read error propagation through readExistingDeck and changeDeck.
	unknownVault := domain.Vault{ID: "unknown"}
	if _, err := e.page.Execute(t.Context(), unknownVault, deckedit.Query{Path: mammals}); err == nil {
		t.Fatal("expected error for unknown vault")
	}
	if _, err := e.editCard.Execute(t.Context(), unknownVault, deckedit.DeckVersion{Path: mammals}, "k7m2xq9fzp", "", nil); err == nil {
		t.Fatal("expected error for unknown vault")
	}
}
