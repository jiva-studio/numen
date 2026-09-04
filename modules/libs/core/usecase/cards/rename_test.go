package cards_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/cardid"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
)

func renaming(t *testing.T, vs vaulted) cards.RenameField {
	t.Helper()
	return cards.RenameField{
		Readers: filesystem.VaultReaders{},
		Writers: filesystem.VaultWriters{},
		Notes:   vs.db.NoteQueries(),
		Links:   vs.db.NoteQueries(),
		Index:   vs.index(t),
	}
}

// A field renamed in a stencil is renamed in every card that stencil cuts, and
// in no card of another one. The value under the heading is left where it was.
func TestAFieldRenamedInAStencilIsRenamedInEveryCardItCuts(t *testing.T) {
	vs := indexed(t)

	got, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Birds.md", "decks/Mammals.md"}) {
		t.Errorf("decks = %v", got.Decks)
	}
	if got.Cards != 2 {
		t.Errorf("cards = %d, want the two cards that stencil cuts", got.Cards)
	}
	if len(got.NotWritten) != 0 {
		t.Errorf("not written = %+v", got.NotWritten)
	}

	stencil := read(t, vs.first, "Animal.md")
	if !strings.Contains(stencil, "fields:\n  - Name\n  - Shoulder height\n  - Life span\n") {
		t.Errorf("the stencil was not renamed: %q", stencil)
	}
	if !strings.Contains(stencil, "mine: keep me verbatim") {
		t.Errorf("a key nobody owns was rewritten: %q", stencil)
	}

	mammals := read(t, vs.first, "decks/Mammals.md")
	if !strings.Contains(mammals, "### Shoulder height\n\nabout 45\"\n") {
		t.Errorf("the card of that stencil was not renamed: %q", mammals)
	}
	// The card of another stencil keeps the heading of that name.
	if !strings.Contains(mammals, "## Gloss ^zpqrstvwxy\n\n[[Term]]\n\n### Height\n\nnot a length at all\n") {
		t.Errorf("a card of another stencil was renamed: %q", mammals)
	}
	if !strings.Contains(mammals, "mine: keep me verbatim") {
		t.Errorf("the deck's own key was rewritten: %q", mammals)
	}
	if !strings.Contains(read(t, vs.first, "decks/Birds.md"), "### Shoulder height\n\nabout 4\"\n") {
		t.Error("the deck nobody had open was not reached")
	}
}

// A card names its stencil by an ordinary wikilink, and reading the deck cuts
// that card by whatever the link lands on. The rename holds to the same answer,
// so a card naming its stencil by anything but the file's own name is reached.
func TestARenameReachesACardWhoseLinkNamesThePath(t *testing.T) {
	vs := indexed(t)
	write(t, vs.first, "cards/Bird.md",
		"---\ntype: stencil\nfields:\n  - Species\n  - Height\n---\n\n"+
			"## Recognise\n\n### Front\n\n{{Species}}\n\n### Back\n\n{{Height}}\n")
	write(t, vs.first, "decks/Wrens.md",
		"---\ntype: deck\n---\n\n## Wren\n\n[[cards/Bird]]\n\n### Height\n\nabout 4\"\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	got, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "cards/Bird.md", From: "Height", To: "Wingspan",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Wrens.md"}) || got.Cards != 1 {
		t.Errorf("decks = %v, cards = %d, want the one card that link lands on", got.Decks, got.Cards)
	}
	if !strings.Contains(read(t, vs.first, "decks/Wrens.md"), "### Wingspan\n\nabout 4\"\n") {
		t.Errorf("the card was not renamed: %q", read(t, vs.first, "decks/Wrens.md"))
	}
}

// The first field stands under a heading of its own in every card, so renaming
// it is a write to every deck that stencil cuts, as renaming any other field
// already was.
func TestRenamingTheFirstFieldReachesEveryDeckThatStencilCuts(t *testing.T) {
	vs := indexed(t)

	got, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Name", To: "Species",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Birds.md", "decks/Mammals.md"}) {
		t.Errorf("decks = %v", got.Decks)
	}
	if got.Cards != 2 {
		t.Errorf("cards = %d, want the two cards that stencil cuts", got.Cards)
	}
	if len(got.NotWritten) != 0 {
		t.Errorf("not written = %+v", got.NotWritten)
	}

	stencil := read(t, vs.first, "Animal.md")
	if !strings.Contains(stencil, "fields:\n  - Species\n  - Height\n") {
		t.Errorf("the stencil was not renamed: %q", stencil)
	}
	// The heading is read from the field, and it stands where it stood: what a
	// field is called is no part of what a card is called.
	mammals := read(t, vs.first, "decks/Mammals.md")
	if !strings.Contains(mammals, "## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Species\n\nLlama\n") {
		t.Errorf("the card of that stencil was not renamed: %q", mammals)
	}
	// The card of another stencil keeps the heading of that name.
	if !strings.Contains(mammals, "### Word\n\nGloss\n") {
		t.Errorf("a card of another stencil was renamed: %q", mammals)
	}
	if !strings.Contains(read(t, vs.first, "decks/Birds.md"), "### Species\n\nWren\n") {
		t.Error("the deck nobody had open was not reached")
	}
}

// A rename reaches the decks of the vault it was asked of, and no other's. The
// second vault holds a card naming the first vault's stencil, under a heading of
// the same name.
func TestARenameStaysInItsOwnVault(t *testing.T) {
	vs := indexed(t)
	quartz := read(t, vs.second, "decks/Quartz.md")

	if _, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if after := read(t, vs.second, "decks/Quartz.md"); after != quartz {
		t.Errorf("the other vault was written\n was %q\n now %q", quartz, after)
	}

	// And the other way round: the second vault's own stencil reaches its own
	// deck, and nothing of the first's.
	mammals := read(t, vs.first, "decks/Mammals.md")
	if _, err := renaming(t, vs).Execute(t.Context(), vs.second, cards.Rename{
		Stencil: "Mineral.md", From: "Height", To: "Crystal habit",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if after := read(t, vs.first, "decks/Mammals.md"); after != mammals {
		t.Errorf("the first vault was written\n was %q\n now %q", mammals, after)
	}
	after := read(t, vs.second, "decks/Quartz.md")
	if !strings.Contains(after, "### Crystal habit\n\na crystal habit\n") {
		t.Errorf("the vault's own deck was not reached: %q", after)
	}
	if !strings.Contains(after, "[[Animal]]\n\n### Height\n\nabout 45\"\n") {
		t.Errorf("the card of the other vault's stencil was renamed: %q", after)
	}
}

// A deck the rename could not be written to keeps the old heading, that is a
// problem against that deck, and the decks after it are written all the same.
func TestADeckTheRenameCouldNotReachKeepsTheOldHeading(t *testing.T) {
	vs := indexed(t)
	before := read(t, vs.first, "decks/Birds.md")

	u := renaming(t, vs)
	u.Writers = refusing{VaultWriters: filesystem.VaultWriters{}, path: "decks/Birds.md"}

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Mammals.md"}) {
		t.Errorf("decks = %v", got.Decks)
	}
	if len(got.NotWritten) != 1 || got.NotWritten[0].Path != "decks/Birds.md" {
		t.Fatalf("not written = %+v", got.NotWritten)
	}
	if got.NotWritten[0].Problem.Fault != format.FaultNotWritten {
		t.Errorf("problem = %+v", got.NotWritten[0].Problem)
	}
	if after := read(t, vs.first, "decks/Birds.md"); after != before {
		t.Errorf("the deck was written\n was %q\n now %q", before, after)
	}
	if !strings.Contains(read(t, vs.first, "decks/Mammals.md"), "### Shoulder height\n") {
		t.Error("the deck after the refused one was not written")
	}
}

// A rename is the application writing a deck, so the deck it leaves behind is
// whole: a card carrying no mark is given one, and a heading standing out of
// step with its first field is written again from it.
func TestARenameMakesTheDeckItWritesWhole(t *testing.T) {
	vs := indexed(t)
	write(t, vs.first, "decks/Hand.md", "---\ntype: deck\n---\n"+
		"\n## Something else\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}

	deck := held(t, vs, "decks/Hand.md")
	if len(deck.Cards) != 1 {
		t.Fatalf("cards = %+v", deck.Cards)
	}
	if !cardid.Valid(deck.Cards[0].Mark) {
		t.Errorf("mark = %q, want the card given one", deck.Cards[0].Mark)
	}
	if deck.Cards[0].Heading != "Llama" {
		t.Errorf("heading = %q, want it written again from the first field", deck.Cards[0].Heading)
	}
}

// The stencil leads. A rename the stencil refused reaches no deck.
func TestARenameTheStencilRefusedReachesNoDeck(t *testing.T) {
	vs := indexed(t)
	before := read(t, vs.first, "decks/Mammals.md")

	_, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Girth", To: "Waist",
	})
	if !errors.Is(err, format.ErrNoSuchField) {
		t.Fatalf("rename = %v, want it refused", err)
	}
	if after := read(t, vs.first, "decks/Mammals.md"); after != before {
		t.Errorf("a deck was written\n was %q\n now %q", before, after)
	}
}

// A deck over the bound is not read, so the rename does not reach it and says
// so.
func TestADeckOverTheBoundIsNotRenamed(t *testing.T) {
	vs := indexed(t)
	if err := os.Truncate(filepath.Join(vs.first.Path, "decks", "Birds.md"), cards.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	got, err := renaming(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if len(got.NotWritten) != 1 || got.NotWritten[0].Path != "decks/Birds.md" {
		t.Fatalf("not written = %+v", got.NotWritten)
	}
	if !strings.Contains(got.NotWritten[0].Problem.Detail, "8388608") {
		t.Errorf("the bound was not said: %q", got.NotWritten[0].Problem.Detail)
	}
}

// byType is the index, watching for the questions a rename must not ask.
type byType struct {
	port.NoteQueries
	t *testing.T
}

func (q byType) Fingerprints(ctx context.Context, vaultID domain.VaultID) (map[string]domain.Fingerprint, error) {
	q.t.Error("a rename asked the index about every file of the vault")
	return q.NoteQueries.Fingerprints(ctx, vaultID)
}

func (q byType) Types(
	ctx context.Context, vaultID domain.VaultID, paths []string,
) (map[string]domain.NoteType, error) {
	q.t.Errorf("a rename asked what each of %d notes is", len(paths))
	return q.NoteQueries.Types(ctx, vaultID, paths)
}

// A rename asks the index for the decks of the vault and for nothing else.
//
// The whole of it runs under the vault's write lock, so a question per note is
// every other write in the application waiting behind a vault-sized loop.
func TestARenameAsksTheIndexForTheDecksAndNotForEveryNote(t *testing.T) {
	vs := indexed(t)
	u := renaming(t, vs)
	u.Notes = byType{NoteQueries: vs.db.NoteQueries(), t: t}

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Birds.md", "decks/Mammals.md"}) {
		t.Errorf("decks = %v", got.Decks)
	}
}
