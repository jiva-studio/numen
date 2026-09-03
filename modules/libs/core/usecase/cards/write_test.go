package cards_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	format "github.com/jiva-studio/numen/modules/libs/core/cards"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/mark"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/cards"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// counted is a writer that says how many times a file was replaced, so a test
// about one write can say it was one.
type counted struct {
	port.VaultWriters
	writes int
}

func (w *counted) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := w.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return &countedWriter{VaultWriter: writer, on: w}, nil
}

type countedWriter struct {
	port.VaultWriter
	on *counted
}

func (w *countedWriter) Write(
	ctx context.Context, path string, content []byte, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	w.on.writes++
	return w.VaultWriter.Write(ctx, path, content, fingerprint)
}

// The faces are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both. Two writes leave the file
// in a shape nobody asked for whenever the second of them does not land.
func TestAStencilsFacesAndItsFieldsAreOneWrite(t *testing.T) {
	vs := indexed(t)
	writers := &counted{VaultWriters: filesystem.VaultWriters{}}
	u := cards.Write{Readers: filesystem.VaultReaders{}, Writers: writers}

	body := "\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Wingspan}}\n"
	at, err := u.Stencil(
		t.Context(), vs.first, "Animal.md", body,
		[]string{"Name", "Wingspan"}, domain.Fingerprint{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if writers.writes != 1 {
		t.Errorf("the stencil was written %d times", writers.writes)
	}

	held := read(t, vs.first, "Animal.md")
	if !strings.Contains(held, "fields:\n  - Name\n  - Wingspan\n") {
		t.Errorf("the stencil declares something else: %q", held)
	}
	if !strings.Contains(held, "{{Wingspan}}") {
		t.Errorf("the faces are not the ones written: %q", held)
	}
	if !strings.Contains(held, "mine: keep me verbatim") {
		t.Errorf("a key nobody owns was rewritten: %q", held)
	}
	// The fingerprint that comes back is the file on disk, so the next write of
	// it lands.
	if at.Size != int64(len(held)) {
		t.Errorf("the fingerprint is not the file: %+v", at)
	}
}

// A stencil already declaring exactly these fields, in this order, is not
// written a second time, so what the person wrote around them stands.
func TestAStencilAlreadyDeclaringTheseFieldsKeepsWhatStandsAroundThem(t *testing.T) {
	vs := indexed(t)
	write(t, vs.first, "Kept.md", "---\ntype: stencil\nfields:\n"+
		"  # the one the card is named by\n  - Name\n  - Height\n---\n"+
		"\n## Recognise\n\n### Front\n\n{{Name}}\n")

	u := cards.Write{Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}}
	if _, err := u.Stencil(
		t.Context(), vs.first, "Kept.md",
		"\n## Recognise\n\n### Front\n\n{{Height}}\n",
		[]string{"Name", "Height"}, domain.Fingerprint{},
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	held := read(t, vs.first, "Kept.md")
	if !strings.Contains(held, "  # the one the card is named by\n") {
		t.Errorf("the fields were written over: %q", held)
	}
	if !strings.Contains(held, "{{Height}}") {
		t.Errorf("the faces are not the ones written: %q", held)
	}
}

// laid puts a deck in the vault and brings the index up to date, so that the
// wikilink under each card's heading reaches the stencil it names.
func laid(t *testing.T, vs vaults, path, body string) cards.Write {
	t.Helper()
	write(t, vs.first, path, "---\ntype: deck\n---\n"+body)
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}
	return cards.Write{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}, Links: vs.db.NoteQueries(),
	}
}

// held is the deck as the vault now holds it.
func held(t *testing.T, vs vaults, path string) format.Deck {
	t.Helper()
	got, err := cards.Read{Readers: filesystem.VaultReaders{}, Links: vs.db.NoteQueries()}.
		Deck(t.Context(), vs.first, path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return got.Deck
}

// A deck goes to disk through the note writer, and is read at MaxBytes. The
// writer carries that bound, so a deck larger than a note is written.
func TestWritingADeckLargerThanANote(t *testing.T) {
	vs := indexed(t)
	body := "\n## Llama\n\n[[Animal]]\n\n### Name\n\n" +
		strings.Repeat("Llama ", note.MaxBytes/6) + "\n"
	if len(body) <= note.MaxBytes {
		t.Fatalf("the deck is %d bytes, and a note is bounded at %d", len(body), note.MaxBytes)
	}
	w := laid(t, vs, "decks/Long.md", body)

	if _, err := w.Deck(t.Context(), vs.first, "decks/Long.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if deck := held(t, vs, "decks/Long.md"); len(deck.Cards) != 1 {
		t.Errorf("cards = %d", len(deck.Cards))
	}
}

// A card typed into a deck by hand carries no mark until the application next
// writes that file, which is when it is given one.
func TestWritingADeckMintsAMarkForEveryCardCarryingNone(t *testing.T) {
	vs := indexed(t)
	body := "\n## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n" +
		"\n## Alpaca\n\n[[Animal]]\n\n### Name\n\nAlpaca\n"
	w := laid(t, vs, "decks/Hand.md", body)

	if _, err := w.Deck(t.Context(), vs.first, "decks/Hand.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}

	deck := held(t, vs, "decks/Hand.md")
	if len(deck.Cards) != 2 {
		t.Fatalf("cards = %+v", deck.Cards)
	}
	for _, card := range deck.Cards {
		if !mark.Valid(card.Mark) {
			t.Errorf("%q carries %q, which is no mark", card.Heading, card.Mark)
		}
	}
	if deck.Cards[0].Mark == deck.Cards[1].Mark {
		t.Errorf("both cards carry %q", deck.Cards[0].Mark)
	}

	// A mark is what the card is for as long as it exists, so the next write
	// leaves it where it stands.
	after := read(t, vs.first, "decks/Hand.md")
	if _, err := w.Deck(
		t.Context(), vs.first, "decks/Hand.md", prose(t, after), domain.Fingerprint{},
	); err != nil {
		t.Fatalf("write again: %v", err)
	}
	if got := read(t, vs.first, "decks/Hand.md"); got != after {
		t.Errorf("the second write moved a mark\n was %q\n now %q", after, got)
	}
}

// A mark is written into a file of carriage returns without disturbing them.
func TestAMarkIsWrittenInTheFilesOwnLineEnding(t *testing.T) {
	vs := indexed(t)
	body := "\n## Llama\n\n[[Animal]]\n\n### Name\n\nLlama\n"
	write(t, vs.first, "decks/Crlf.md",
		strings.ReplaceAll("---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: deck\n---\n"+body, "\n", "\r\n"))
	if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
		t.Fatal(err)
	}

	w := cards.Write{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}, Links: vs.db.NoteQueries(),
	}
	if _, err := w.Deck(t.Context(), vs.first, "decks/Crlf.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := read(t, vs.first, "decks/Crlf.md")
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("a bare break was written into a file of carriage returns: %q", got)
	}
	if !mark.Valid(held(t, vs, "decks/Crlf.md").Cards[0].Mark) {
		t.Errorf("no mark was written: %q", got)
	}
}

// The field is what stands. A person may leave the heading disagreeing with it,
// and the next write puts the heading back in step.
func TestWritingADeckPutsAStaleHeadingBackInStep(t *testing.T) {
	vs := indexed(t)
	body := "\n## Something else ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n## Also stale ^zpqrstvwxy\n\n[[Animal]]\n\n### Name\n" +
		"\n### Height\n\nthe first field is a box to fill\n"
	w := laid(t, vs, "decks/Stale.md", body)

	if _, err := w.Deck(t.Context(), vs.first, "decks/Stale.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}

	deck := held(t, vs, "decks/Stale.md")
	if deck.Cards[0].Heading != "Llama" || deck.Cards[0].Mark != "k7m2xq9fzp" {
		t.Errorf("card = %+v, want the heading written again from the field", deck.Cards[0])
	}
	// An empty first field gives a heading of nothing, and the card stands under
	// its mark alone.
	if deck.Cards[1].Heading != "" || deck.Cards[1].Mark != "zpqrstvwxy" {
		t.Errorf("card = %+v, want a heading of nothing", deck.Cards[1])
	}
	if got := read(t, vs.first, "decks/Stale.md"); !strings.Contains(got, "## ^zpqrstvwxy\n") {
		t.Errorf("heading written wrong: %q", got)
	}
}

// A card carrying no heading at all for the field that is first says nothing
// about what its heading should be, so the heading is left exactly as it
// stands. A rename that could not be written to a deck leaves every card in it
// that way, and the next ordinary save must not read that as a heading of
// nothing.
func TestACardWritingNothingUnderTheFirstFieldKeepsItsHeading(t *testing.T) {
	vs := indexed(t)
	// The stencil declares Name first, and this card was written when the field
	// standing first was called something else.
	body := "\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Was called something else\n\nLlama\n"
	w := laid(t, vs, "decks/Behind.md", body)

	if _, err := w.Deck(t.Context(), vs.first, "decks/Behind.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}

	deck := held(t, vs, "decks/Behind.md")
	if deck.Cards[0].Heading != "Llama" {
		t.Errorf("heading = %q, want it left as it stands", deck.Cards[0].Heading)
	}
}

// The mark is minted where the deck is made whole, so the write is what knows
// it, and a caller that has just added a card is told what to address it by.
func TestAWriteSaysWhichMarksItMinted(t *testing.T) {
	vs := indexed(t)
	body := "\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n" +
		"\n## Alpaca\n\n[[Animal]]\n\n### Name\n\nAlpaca\n"
	w := laid(t, vs, "decks/Minted.md", body)

	wrote, err := w.Deck(t.Context(), vs.first, "decks/Minted.md", body, domain.Fingerprint{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(wrote.Minted) != 1 || wrote.Minted[0].Card != 1 {
		t.Fatalf("minted = %+v, want the one card that carried none", wrote.Minted)
	}
	if got := held(t, vs, "decks/Minted.md").Cards[1].Mark; got != wrote.Minted[0].Mark {
		t.Errorf("the mark reported is %q and the card carries %q", wrote.Minted[0].Mark, got)
	}
}

// Nothing can say which field is first where the stencil cannot be read, so the
// heading is left exactly as it stands. The card is given its mark all the same.
func TestACardWhoseStencilCannotBeReadIsNotReprojected(t *testing.T) {
	vs := indexed(t)
	body := "\n## Whatever a person typed\n\n[[Nowhere]]\n\n### Name\n\nLlama\n"
	w := laid(t, vs, "decks/Loose.md", body)

	if _, err := w.Deck(t.Context(), vs.first, "decks/Loose.md", body, domain.Fingerprint{}); err != nil {
		t.Fatalf("write: %v", err)
	}

	deck := held(t, vs, "decks/Loose.md")
	if deck.Cards[0].Heading != "Whatever a person typed" {
		t.Errorf("heading = %q, want it left as it stands", deck.Cards[0].Heading)
	}
	if !mark.Valid(deck.Cards[0].Mark) {
		t.Errorf("mark = %q", deck.Cards[0].Mark)
	}
}

// A deck whose cards are already whole comes back the file it was, under either
// line ending. Anything less is a diff nobody asked for, on every save, forever.
func TestWritingADeckNobodyTouchedChangesNothing(t *testing.T) {
	body := "\nCards I am learning.\n" +
		"\n## Llama ^k7m2xq9fzp\n\n[[Animal]]\n\n### Name\n\nLlama\n\n### Height\n\nabout 45\"\n" +
		"\n## ^zpqrstvwxy\n\n[[Animal]]\n\n### Name\n\n"
	for name, ending := range map[string]string{"lf": "\n", "crlf": "\r\n"} {
		t.Run(name, func(t *testing.T) {
			vs := indexed(t)
			raw := strings.ReplaceAll(
				"---\nid: 01J8F3K2M9QRSTVWXYZ012\ntype: deck\n---\n"+body, "\n", ending)
			write(t, vs.first, "decks/Whole.md", raw)
			if err := vs.index(t)(t.Context(), vs.first, nil); err != nil {
				t.Fatal(err)
			}

			w := cards.Write{
				Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{},
				Links: vs.db.NoteQueries(),
			}
			if _, err := w.Deck(
				t.Context(), vs.first, "decks/Whole.md", body, domain.Fingerprint{},
			); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := read(t, vs.first, "decks/Whole.md"); got != raw {
				t.Errorf("the file changed\n was %q\n now %q", raw, got)
			}
		})
	}
}
