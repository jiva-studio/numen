package cards_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cardid"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/cards"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/format"
	"github.com/jiva-studio/numen/modules/plugins/views.flashcards.studio/backend/ports"
)

func newRenameField(t *testing.T, vs vaulted) cards.RenameField {
	t.Helper()
	return cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		vs.queries, vs.queries, vs.index(t), time.Now, cards.UnboundedDecks,
	)
}

// A field renamed in a stencil is renamed in every card that stencil cuts, and
// in no card of another one. The value under the heading is left where it was.
func TestAFieldRenamedInAStencilIsRenamedInEveryCardItSplits(t *testing.T) {
	vs := newVaults(t)

	got, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
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
	vs := newVaults(t)
	write(t, vs.first, "cards/Bird.md",
		"---\ntype: stencil\nfields:\n  - Species\n  - Height\n---\n\n"+
			"## Recognise\n\n### Front\n\n{{Species}}\n\n### Back\n\n{{Height}}\n")
	write(t, vs.first, "decks/Wrens.md",
		"---\ntype: deck\n---\n\n## Wren\n\n[[cards/Bird]]\n\n### Height\n\nabout 4\"\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	got, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
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

// The first field is placed under a heading of its own in every card, so renaming
// it is a write to every deck that stencil cuts, as renaming any other field
// already was.
func TestRenamingTheFirstFieldReachesEveryDeckThatStencilSplits(t *testing.T) {
	vs := newVaults(t)

	got, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
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
	vs := newVaults(t)
	quartz := read(t, vs.second, "decks/Quartz.md")

	if _, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
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
	if _, err := newRenameField(t, vs).Execute(t.Context(), vs.second, cards.Rename{
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
	vs := newVaults(t)
	before := read(t, vs.first, "decks/Birds.md")

	u := newRenameField(t, vs).WithWriters(refusing{VaultWriters: vs.ports.Writers, path: "decks/Birds.md"})

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
		t.Error("the deck after the rejected one was not written")
	}
}

// whole: a card carrying no mark is given one, and a heading standing out of
// step with its first field is written again from it.
func TestARenameMakesTheDeckItWritesWhole(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "decks/Hand.md", "---\ntype: deck\n---\n"+
		"\n## Something else\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n")
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	if _, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}

	deck := readDeck(t, vs, "decks/Hand.md")
	if len(deck.Cards) != 1 {
		t.Fatalf("cards = %+v", deck.Cards)
	}
	if !cardid.IsValid(deck.Cards[0].Mark) {
		t.Errorf("mark = %q, want the card given one", deck.Cards[0].Mark)
	}
	if deck.Cards[0].Heading != "Llama" {
		t.Errorf("heading = %q, want it written again from the first field", deck.Cards[0].Heading)
	}
}

func TestARenameTheStencilRefusedReachesNoDeck(t *testing.T) {
	vs := newVaults(t)
	before := read(t, vs.first, "decks/Mammals.md")

	_, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Girth", To: "Waist",
	})
	if !errors.Is(err, format.ErrNoSuchField) {
		t.Fatalf("rename = %v, want it rejected", err)
	}
	if after := read(t, vs.first, "decks/Mammals.md"); after != before {
		t.Errorf("a deck was written\n was %q\n now %q", before, after)
	}
}

// A deck over the bound is not read, so the rename does not reach it and says
// so.
func TestADeckOverTheBoundIsNotRenamed(t *testing.T) {
	vs := newVaults(t)
	if err := os.Truncate(filepath.Join(vs.first.Path, "decks", "Birds.md"), cards.MaxBytes+1); err != nil {
		t.Fatal(err)
	}

	got, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
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

// byType is the index, watching for the question a rename must not ask and
// counting the one it may.
type byType struct {
	ports.NoteQueries
	t     *testing.T
	queryCount *int
}

func (q byType) GetPathsOfType(
	ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType,
) ([]string, error) {
	*q.queryCount++
	return q.NoteQueries.GetPathsOfType(ctx, vaultID, noteType)
}

func (q byType) GetTypes(
	ctx context.Context, vaultID domain.VaultID, paths []string,
) (map[string]domain.NoteType, error) {
	q.t.Errorf("a rename asked what each of %d notes is", len(paths))
	return q.NoteQueries.GetTypes(ctx, vaultID, paths)
}

// A rename asks the index for the decks of the vault and for nothing else.
//
// The whole of it runs under the vault's write lock, so a question per note is
// every other write in the application waiting behind a vault-sized loop.
func TestARenameAsksTheIndexForTheDecksAndNotForEveryNote(t *testing.T) {
	vs := newVaults(t)
	var request int
	u := newRenameField(t, vs).WithNotes(byType{NoteQueries: vs.queries, t: t, queryCount: &request})

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Birds.md", "decks/Mammals.md"}) {
		t.Errorf("decks = %v", got.Decks)
	}
	if request != 1 {
		t.Errorf("the index was asked for the decks %d times, want once", request)
	}
}

// shut is a vault that refuses to be written: no lock is given and no writer
// opens. A test that marks it watched fails on being asked at all.
type shut struct {
	ports.VaultWriters
	t         *testing.T
	isWatched bool
	hold      error
	open      error
}

func (s shut) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	if s.isWatched {
		s.t.Error("a rename took the write lock")
	}
	if s.hold != nil {
		return nil, s.hold
	}
	return s.VaultWriters.Lock(ctx, v)
}

func (s shut) Open(v domain.Vault) (ports.VaultWriter, error) {
	if s.isWatched {
		s.t.Error("a rename opened the vault for writing")
	}
	if s.open != nil {
		return nil, s.open
	}
	return s.VaultWriters.Open(v)
}

// decksOf is an index that names the decks of a vault from a list, or fails.
type decksOf struct {
	paths []string
	err   error
	// cancel is called once the decks have been named.
	cancel func()
}

func (d decksOf) GetPathsOfType(context.Context, domain.VaultID, domain.NoteType) ([]string, error) {
	if d.cancel != nil {
		d.cancel()
	}
	return d.paths, d.err
}

// A field renamed to its own name is no rename, and nothing is written.
func TestRenamingAFieldToItsOwnNameWritesNothing(t *testing.T) {
	vs := newVaults(t)
	stencil, err := cards.NewRead(vs.ports.Readers, nil).Stencil(t.Context(), vs.first, "Animal.md")
	if err != nil {
		t.Fatal(err)
	}
	u := newRenameField(t, vs).WithWriters(shut{t: t, isWatched: true, hold: errOutOfReach, open: errOutOfReach})

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Height", Fingerprint: stencil.Fingerprint,
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if got.Stencil != stencil.Fingerprint || len(got.Decks) != 0 {
		t.Errorf("result = %+v, want the stencil as the caller read it and no deck", got)
	}
}

// Every file a rename wrote is brought level in the index: the stencil, and each
// deck a heading was rewritten in.
func TestARenameLevelsEveryFileItWrote(t *testing.T) {
	vs := newVaults(t)
	var levelled []string
	u := cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		vs.queries, vs.queries,
		func(_ context.Context, _ domain.Vault, paths []string) error {
			levelled = append(levelled, paths...)
			return nil
		},
		time.Now, cards.UnboundedDecks,
	)

	if _, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(levelled, []string{"Animal.md", "decks/Birds.md", "decks/Mammals.md"}) {
		t.Errorf("levelled = %v", levelled)
	}
}

// A rename made with nothing to level the index writes all the same, and says
// what it wrote.
func TestARenameWithNoIndexToLevelReturnsWhatItWrote(t *testing.T) {
	vs := newVaults(t)
	u := cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		vs.queries, vs.queries, nil, time.Now, cards.UnboundedDecks,
	)

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Birds.md", "decks/Mammals.md"}) || got.Cards != 2 {
		t.Errorf("decks = %v, cards = %d", got.Decks, got.Cards)
	}
	if got.Stencil.IsZero() {
		t.Error("the stencil's new fingerprint was not said")
	}
}

// A rename is the application writing, so a file it writes that carried no
// identifier carries one afterwards.
func TestARenameProducesEveryFileItWritesAnIdentifier(t *testing.T) {
	vs := newVaults(t)
	for _, path := range []string{"Animal.md", "decks/Birds.md"} {
		if strings.Contains(read(t, vs.first, path), "\nid: ") {
			t.Fatalf("the fixture %s already carries an identifier", path)
		}
	}

	if _, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	for _, path := range []string{"Animal.md", "decks/Birds.md"} {
		if raw := read(t, vs.first, path); !strings.Contains(raw, "\nid: ") {
			t.Errorf("%s carries no identifier: %q", path, raw)
		}
	}
}

// A vault the rename cannot lock, read or write is an error, and nothing is
// written.
func TestARenameInAVaultOutOfReachWritesNothing(t *testing.T) {
	vs := newVaults(t)
	for _, c := range []struct {
		name    string
		readers ports.VaultReaders
		writers ports.VaultWriters
	}{
		{"no lock", vs.ports.Readers, shut{VaultWriters: vs.ports.Writers, hold: errOutOfReach}},
		{"no reader", faulty{inner: vs.ports.Readers, open: errOutOfReach}, vs.ports.Writers},
		{"no writer", vs.ports.Readers, shut{VaultWriters: vs.ports.Writers, open: errOutOfReach}},
	} {
		t.Run(c.name, func(t *testing.T) {
			stencil, birds := read(t, vs.first, "Animal.md"), read(t, vs.first, "decks/Birds.md")
			u := cards.NewRenameField(
				c.readers, c.writers, vs.queries, vs.queries, indexNothing, time.Now, cards.UnboundedDecks,
			)

			_, err := u.Execute(t.Context(), vs.first, cards.Rename{
				Stencil: "Animal.md", From: "Height", To: "Shoulder height",
			})
			if !errors.Is(err, errOutOfReach) {
				t.Fatalf("rename = %v, want the vault's error", err)
			}
			if read(t, vs.first, "Animal.md") != stencil || read(t, vs.first, "decks/Birds.md") != birds {
				t.Error("a file was written")
			}
		})
	}
}

// A stencil the vault cannot look at or read is an error, and no file is
// written.
func TestARenameOfAStencilOutOfReachWritesNothing(t *testing.T) {
	vs := newVaults(t)
	for _, c := range []struct {
		name    string
		readers faulty
	}{
		{"not looked at", faulty{inner: vs.ports.Readers, stat: map[string]error{"Animal.md": errOutOfReach}}},
		{"not read", faulty{inner: vs.ports.Readers, read: map[string]error{"Animal.md": errOutOfReach}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			stencil, birds := read(t, vs.first, "Animal.md"), read(t, vs.first, "decks/Birds.md")
			u := cards.NewRenameField(
				c.readers, vs.ports.Writers, vs.queries, vs.queries, indexNothing, time.Now, cards.UnboundedDecks,
			)

			_, err := u.Execute(t.Context(), vs.first, cards.Rename{
				Stencil: "Animal.md", From: "Height", To: "Shoulder height",
			})
			if !errors.Is(err, errOutOfReach) {
				t.Fatalf("rename = %v, want the stencil's error", err)
			}
			if read(t, vs.first, "Animal.md") != stencil || read(t, vs.first, "decks/Birds.md") != birds {
				t.Error("a file was written")
			}
		})
	}
}

// A stencil whose frontmatter is not YAML fails, and no deck is written.
func TestARenameOfAStencilThatWillNotOpenWritesNothing(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "Animal.md", "---\nfields: [Name\n---\n\n## Recognise\n")
	birds := read(t, vs.first, "decks/Birds.md")

	_, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, markdown.ErrUnreadable) {
		t.Fatalf("rename = %v, want the stencil rejected", err)
	}
	if read(t, vs.first, "decks/Birds.md") != birds {
		t.Error("a deck was written")
	}
}

// The stencil is written against what the vault said it was when the rename
// looked at it, so a person saving it in the meantime keeps what they saved.
func TestARenameDoesNotWriteOverAStencilSavedWhileItRan(t *testing.T) {
	vs := newVaults(t)
	saved := animals["Animal.md"] + "\nA line written while the rename ran.\n"
	readers := faulty{inner: vs.ports.Readers, saved: map[string]string{"Animal.md": saved}, root: vs.first.Path}
	u := cards.NewRenameField(
		readers, vs.ports.Writers, vs.queries, vs.queries, indexNothing, time.Now, cards.UnboundedDecks,
	)

	_, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, ports.ErrStale) {
		t.Fatalf("rename = %v, want it rejected as stale", err)
	}
	if after := read(t, vs.first, "Animal.md"); after != saved {
		t.Errorf("what the person saved was written over: %q", after)
	}
}

// The decks are named by the index, and an index that cannot name them is an
// error of the rename.
func TestARenameWhoseDecksCannotBeNamedIsAnError(t *testing.T) {
	vs := newVaults(t)
	birds := read(t, vs.first, "decks/Birds.md")
	u := newRenameField(t, vs).WithNotes(decksOf{err: errOutOfReach})

	_, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("rename = %v, want the index's error", err)
	}
	if read(t, vs.first, "decks/Birds.md") != birds {
		t.Error("a deck was written")
	}
}

// A rename whose caller has gone stops before the next deck.
func TestARenameWhoseContextEndsWritesNoFurtherDeck(t *testing.T) {
	vs := newVaults(t)
	birds, mammals := read(t, vs.first, "decks/Birds.md"), read(t, vs.first, "decks/Mammals.md")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	u := cards.NewRenameField(
		vs.ports.Readers, vs.ports.Writers,
		decksOf{paths: []string{"decks/Birds.md", "decks/Mammals.md"}, cancel: cancel},
		vs.queries, indexNothing, time.Now, cards.UnboundedDecks,
	)

	got, err := u.Execute(ctx, vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("rename = %v, want the caller's end", err)
	}
	if len(got.Decks) != 0 || len(got.NotWritten) != 0 {
		t.Errorf("result = %+v, want no deck reached", got)
	}
	if read(t, vs.first, "decks/Birds.md") != birds || read(t, vs.first, "decks/Mammals.md") != mammals {
		t.Error("a deck was written")
	}
}

// A deck the rename cannot reach keeps what it holds and is reported, whatever
// of another of its cards.
func TestADeckOutOfReachIsReportedAndLeftAsItWas(t *testing.T) {
	for _, c := range []struct {
		name    string
		path    string
		readers ports.VaultReaders
		links   ports.LinkQueries
		// written is what the deck holds by the time the rename runs, after the
		// index read it as a deck.
		content string
	}{
		{name: "not looked at", path: "decks/Birds.md",
			readers: faulty{stat: map[string]error{"decks/Birds.md": errOutOfReach}}},
		{name: "not read", path: "decks/Birds.md",
			readers: faulty{read: map[string]error{"decks/Birds.md": errOutOfReach}}},
		{name: "links unanswered", path: "decks/Birds.md",
			links: unanswered{}},
		{name: "frontmatter not YAML", path: "decks/Birds.md",
			content: "---\ntype: [deck\n---\n\n## Wren ^3f4g5h6j7k\n\n[[Animal]]\n\n### Height\n\nabout 4\"\n"},
		{name: "frontmatter not closed", path: "decks/Birds.md",
			content: "---\ntype: deck\n\n## Wren ^3f4g5h6j7k\n\n[[Animal]]\n\n### Height\n\nabout 4\"\n"},
		{name: "frontmatter on one line", path: "decks/Birds.md",
			content: "---\n{type: deck}\n---\n\n## Wren ^3f4g5h6j7k\n\n[[Animal]]\n\n### Name\n\nWren\n\n### Height\n\nabout 4\"\n"},
		{name: "another stencil out of reach", path: "decks/Mammals.md",
			readers: faulty{stat: map[string]error{"Term.md": errOutOfReach}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			vs := newVaults(t)
			if c.content != "" {
				write(t, vs.first, c.path, c.content)
			}
			before := read(t, vs.first, c.path)
			readers, links := c.readers, ports.LinkQueries(vs.queries)
			if readers == nil {
				readers = vs.ports.Readers
			}
			if broken, isFaulty := readers.(faulty); isFaulty {
				broken.inner = vs.ports.Readers
				readers = broken
			}
			if c.links != nil {
				links = c.links
			}
			u := cards.NewRenameField(
				readers, vs.ports.Writers, vs.queries, links, indexNothing, time.Now, cards.UnboundedDecks,
			)

			got, err := u.Execute(t.Context(), vs.first, cards.Rename{
				Stencil: "Animal.md", From: "Height", To: "Shoulder height",
			})
			if err != nil {
				t.Fatalf("rename: %v", err)
			}
			var reported bool
			for _, unwritten := range got.NotWritten {
				reported = reported || unwritten.Path == c.path
			}
			if !reported || slices.Contains(got.Decks, c.path) {
				t.Errorf("decks = %v, not written = %+v, want %s reported", got.Decks, got.NotWritten, c.path)
			}
			if after := read(t, vs.first, c.path); after != before {
				t.Errorf("the deck was written\n was %q\n now %q", before, after)
			}
		})
	}
}

// A deck exactly at the bound is read and renamed.
func TestADeckAtTheBoundIsRenamed(t *testing.T) {
	vs := newVaults(t)
	head := "---\ntype: deck\n---\n\n"
	card := "\n\n## Wren\n\n[[Animal]]\n\n### Name\n\nWren\n\n### Height\n\nabout 4\"\n"
	write(t, vs.first, "decks/Large.md", head+strings.Repeat("x", cards.MaxBytes-len(head)-len(card))+card)
	u := newRenameField(t, vs).WithNotes(decksOf{paths: []string{"decks/Large.md"}})

	got, err := u.Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !slices.Equal(got.Decks, []string{"decks/Large.md"}) || got.Cards != 1 || len(got.NotWritten) != 0 {
		t.Errorf("result = %+v, want the one card renamed", got)
	}
}

// A deck holding no card of the stencil is not written, byte for byte, though a
// write would have given its card a mark and the deck an identifier.
func TestADeckContainingNoCardOfTheStencilIsLeftAsItWas(t *testing.T) {
	vs := newVaults(t)
	words := "---\ntype: deck\n---\n\n## Gloss\n\n[[Term]]\n\n### Word\n\nGloss\n\n### Height\n\nnot a length\n"
	write(t, vs.first, "decks/Words.md", words)
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	got, err := newRenameField(t, vs).Execute(t.Context(), vs.first, cards.Rename{
		Stencil: "Animal.md", From: "Height", To: "Shoulder height",
	})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if slices.Contains(got.Decks, "decks/Words.md") {
		t.Errorf("decks = %v", got.Decks)
	}
	if after := read(t, vs.first, "decks/Words.md"); after != words {
		t.Errorf("the deck was written\n was %q\n now %q", words, after)
	}
}
