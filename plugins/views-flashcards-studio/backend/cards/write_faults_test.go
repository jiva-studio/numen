package cards_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/vaulttest"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/cards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/note"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

const wrenDeck = "\n## Wren\n\n[[Animal]]\n\n### Name\n\nWren\n"

func TestADeckOverTheBoundIsNotSaved(t *testing.T) {
	vs := newVaults(t)
	w := newWriteForDeck(t, vs, "decks/Wrens.md", wrenDeck)
	before := read(t, vs.first, "decks/Wrens.md")

	_, err := w.Deck(t.Context(), vs.first, "decks/Wrens.md",
		wrenDeck+strings.Repeat("a", cards.MaxBytes), domain.FileStat{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("write = %v, want too large", err)
	}
	if after := read(t, vs.first, "decks/Wrens.md"); after != before {
		t.Error("the deck was written")
	}
}

func TestACardModelOverTheBoundOfANoteIsNotSaved(t *testing.T) {
	vs := newVaults(t)
	w := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, vaulttest.NewClock())
	before := read(t, vs.first, "Animal.md")

	_, err := w.CardModel(t.Context(), vs.first, "Animal.md",
		"\n## Recognise\n\n"+strings.Repeat("a", note.MaxBytes), []string{"Name"}, domain.FileStat{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("write = %v, want too large", err)
	}
	if after := read(t, vs.first, "Animal.md"); after != before {
		t.Error("the card model was written")
	}
}

// A deck whose card models cannot be found is an error, and the file is left as it
// was.
func TestADeckWhoseLinksCannotBeResolvedIsNotSaved(t *testing.T) {
	vs := newVaults(t)
	newWriteForDeck(t, vs, "decks/Wrens.md", wrenDeck)
	before := read(t, vs.first, "decks/Wrens.md")
	w := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, unanswered{}, indexNothing, vaulttest.NewClock())

	_, err := w.Deck(t.Context(), vs.first, "decks/Wrens.md", wrenDeck+"\n## Crow\n\n[[Animal]]\n", domain.FileStat{})
	if !errors.Is(err, errOutOfReach) {
		t.Fatalf("write = %v, want the index's error", err)
	}
	if after := read(t, vs.first, "decks/Wrens.md"); after != before {
		t.Error("the deck was written")
	}
}

// A write over a file that changed since it was read returns nothing: no
// file stat, and no card id for a card that was never written.
func TestAStaleWritePassesBackNothing(t *testing.T) {
	vs := newVaults(t)
	w := newWriteForDeck(t, vs, "decks/Wrens.md", wrenDeck)
	observed, err := cards.NewRead(vs.ports.Readers, vs.queries).
		Deck(t.Context(), vs.first, "decks/Wrens.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	write(t, vs.first, "decks/Wrens.md", "---\ntype: deck\n---\n"+wrenDeck+"\nChanged under the reader.\n")

	got, err := w.Deck(t.Context(), vs.first, "decks/Wrens.md", wrenDeck, observed.FileStat)
	if !errors.Is(err, ports.ErrStale) {
		t.Fatalf("write = %v, want stale", err)
	}
	if !got.FileStat.IsZero() || len(got.Given) != 0 {
		t.Errorf("result = %+v, want nothing", got)
	}
}

// A deck written to disk whose index could not be brought level is an error
func TestADeckSavedAndNotLevelledPassesBackWhatItExistsAt(t *testing.T) {
	vs := newVaults(t)
	w := newWriteForDeck(t, vs, "decks/Wrens.md", wrenDeck).
		WithIndex(func(_ context.Context, _ domain.Vault, _ []string) error { return errOutOfReach })

	got, err := w.Deck(t.Context(), vs.first, "decks/Wrens.md", wrenDeck, domain.FileStat{})
	if !errors.Is(err, note.ErrUnlevelled) || !errors.Is(err, errOutOfReach) {
		t.Fatalf("write = %v, want the file written and the index not level", err)
	}
	if got.FileStat.Path != "decks/Wrens.md" || got.FileStat.Size == 0 {
		t.Errorf("fingerprint = %+v, want current file state", got.FileStat)
	}
	if len(got.Given) != 1 {
		t.Errorf("given = %+v, want the card id of the one card", got.Given)
	}
	if !strings.Contains(read(t, vs.first, "decks/Wrens.md"), "## Wren ^") {
		t.Error("the deck was not written")
	}
}

// A body opening with the frontmatter delimiter would be read as the card model's
func TestACardModelBodyOpeningWithFrontmatterIsNotSaved(t *testing.T) {
	vs := newVaults(t)
	w := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, vaulttest.NewClock())
	before := read(t, vs.first, "Animal.md")

	_, err := w.CardModel(t.Context(), vs.first, "Animal.md", "---\ntype: deck\n---\n", []string{"Name"}, domain.FileStat{})
	if !errors.Is(err, note.ErrBodyUnwritable) {
		t.Fatalf("write = %v, want the body unwritable", err)
	}
	if after := read(t, vs.first, "Animal.md"); after != before {
		t.Error("the card model was written")
	}
}

// A card model whose frontmatter is never closed has no body to replace, and the
// file is left as it was.
func TestACardModelWhoseFrontmatterIsNeverClosedIsNotSaved(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "Open.md", "---\ntype: card-model\nfields:\n  - Name\n\n## Recognise\n")
	before := read(t, vs.first, "Open.md")
	w := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, vaulttest.NewClock())

	_, err := w.CardModel(t.Context(), vs.first, "Open.md", "\n## Spot it\n", []string{"Sample"}, domain.FileStat{})
	if !errors.Is(err, markdown.ErrUnterminated) {
		t.Fatalf("write = %v, want the frontmatter unterminated", err)
	}
	if after := read(t, vs.first, "Open.md"); after != before {
		t.Error("the card model was written")
	}
}

// written are the ones it declares.
func TestWriteCardModel_ErrorsOnUnterminatedFrontmatter(t *testing.T) {
	vs := newVaults(t)
	write(t, vs.first, "Open.md", "---\ntype: card-model\nfields:\n  - Name\n\n## Recognise\n")
	before := read(t, vs.first, "Open.md")
	w := cards.NewWrite(
		vs.ports.Readers, vs.ports.Writers, vs.queries, indexNothing, vaulttest.NewClock())

	_, err := w.CardModel(t.Context(), vs.first, "Open.md", "\n## Spot it\n", []string{"Name"}, domain.FileStat{})
	if !errors.Is(err, markdown.ErrUnterminated) {
		t.Fatalf("write = %v, want the frontmatter unterminated", err)
	}
	if after := read(t, vs.first, "Open.md"); after != before {
		t.Error("the card model was written")
	}
}
