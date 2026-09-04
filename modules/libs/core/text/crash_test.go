package text_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// wild is a document library given a file it cannot survive. A real one is
// compiled from C and reads bytes whoever synced the vault put there.
type wild struct{}

func (wild) Read(context.Context, []byte) (port.Reading, error) {
	panic("page 3 of 2")
}

func (wild) Lit(context.Context, []byte, []int, []int) ([]highlight.Box, error) {
	panic("page 3 of 2")
}

func (wild) Draw(context.Context, []byte) (port.OpenDocument, error) {
	panic("page 3 of 2")
}

// One book the library dies on is one book. The scan says so and reads the next
// file; the process the person's window runs in is still there to be told.
func TestABookTheLibraryDiesOnIsOneUnreadableFile(t *testing.T) {
	crafted := domain.Fingerprint{Path: "books/atlas.pdf", Kind: domain.KindBook}
	doc, err := text.Read(t.Context(), wild{}, crafted, []byte("%PDF-1.7"))
	if doc != nil {
		t.Errorf("a book that could not be read came back as %+v", doc)
	}
	if !errors.Is(err, text.ErrUnreadable) {
		t.Fatalf("reading it answered %v, want %v", err, text.ErrUnreadable)
	}
	// What was raised is in the error: a boundary that says nothing hides the
	// fault it caught.
	if !strings.Contains(err.Error(), "page 3 of 2") {
		t.Errorf("the error says %q, and not what was raised", err)
	}

	next := domain.Fingerprint{Path: "notes/kepler.md", Kind: domain.KindNote}
	switch read, err := text.Read(t.Context(), wild{}, next, []byte("orbits")); {
	case err != nil:
		t.Fatalf("the next file was not read: %v", err)
	case read.Text != "orbits":
		t.Errorf("the next file says %q", read.Text)
	}
}
