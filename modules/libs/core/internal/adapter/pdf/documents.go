package pdf

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/highlight"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Documents is this library, as the core asks for one.
type Documents struct{}

// Read is what one document says, with the parts it names and where each of
// its pages begins.
func (Documents) Read(ctx context.Context, raw []byte) (port.Reading, error) {
	if err := ctx.Err(); err != nil {
		return port.Reading{}, err
	}
	book, err := Read(raw)
	if err != nil {
		return port.Reading{}, refused(err)
	}
	out := port.Reading{Text: book.Text}
	for _, p := range book.Parts {
		out.Parts = append(out.Parts, chunking.PartStart{Title: p.Title, Offset: p.Offset})
	}
	for _, p := range book.Pages {
		out.Pages = append(out.Pages, p.Offset)
	}
	return out, nil
}

// Lit is where the words of the pages named sit on them.
func (Documents) Lit(ctx context.Context, raw []byte, starts []int, pages []int) ([]highlight.Box, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	book := &Book{Pages: make([]Page, 0, len(starts))}
	for _, at := range starts {
		book.Pages = append(book.Pages, Page{Offset: at})
	}
	boxes, err := book.Lit(raw, pages)
	if err != nil {
		return nil, refused(err)
	}
	return boxes, nil
}

// Draw holds a document open for its pages to be drawn.
func (Documents) Draw(ctx context.Context, raw []byte) (port.OpenDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	scan, err := Open(raw)
	if err != nil {
		return nil, refused(err)
	}
	return scan, nil
}

// refused says which of the two ways a document could not be read, in the words
// the core knows them by.
func refused(err error) error {
	switch {
	case errors.Is(err, ErrEncrypted):
		return fmt.Errorf("%w: %w", port.ErrEncrypted, err)
	case errors.Is(err, ErrNotPDF):
		return fmt.Errorf("%w: %w", port.ErrNotADocument, err)
	}
	return err
}
