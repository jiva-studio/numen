package pdf

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Documents is this library, as the core asks for one.
type Documents struct{}

// Read is what one document says, with the parts it names and where each of
// its pages begins.
func (Documents) Read(ctx context.Context, raw []byte) (out domain.TextLayer, err error) {
	defer recoverPanic("reading a document", &out, &err)
	if err := ctx.Err(); err != nil {
		return domain.TextLayer{}, err
	}
	book, err := Read(raw)
	if err != nil {
		return domain.TextLayer{}, mapReadError(err)
	}
	out = domain.TextLayer{Text: book.Text}
	for _, p := range book.Parts {
		out.Parts = append(out.Parts, domain.PartStart{Title: p.Title, Offset: p.Offset})
	}
	for _, p := range book.Pages {
		out.Pages = append(out.Pages, p.Offset)
	}
	return out, nil
}

// GetHighlights is where the words of the pages named sit on them.
func (Documents) GetHighlights(ctx context.Context, raw []byte, starts []int, pages []int) (boxes []domain.HighlightBox, err error) {
	defer recoverPanic("highlighting a page", &boxes, &err)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	book := &Book{Pages: make([]Page, 0, len(starts))}
	for _, at := range starts {
		book.Pages = append(book.Pages, Page{Offset: at})
	}
	found, err := book.Highlights(raw, pages)
	if err != nil {
		return nil, mapReadError(err)
	}
	return found, nil
}

// OpenDocument holds a document open for its pages to be drawn. It owns the file from
// the call on.
func (Documents) OpenDocument(ctx context.Context, file io.ReadSeekCloser) (open port.OpenDocument, err error) {
	defer recoverPanic("opening a document to draw", &open, &err)
	if err := ctx.Err(); err != nil {
		file.Close()
		return nil, err
	}
	scan, err := OpenReader(file)
	if err != nil {
		return nil, mapReadError(err)
	}
	return scan, nil
}

// recoverPanic is the boundary around the library, deferred by everything that
// hands it a file: a panic raised inside becomes an error about the item that
// raised it, and the answer is the zero one.
//
// A book in a synced vault was put there by whoever synced it, and the library
// reading it is compiled from C. One document, or one page of it, that the
// library cannot survive is worth one error; it is not worth the process the
// person's window runs in.
func recoverPanic[T any](what string, answer *T, err *error) {
	if raised := recover(); raised != nil {
		var none T
		*answer, *err = none, fmt.Errorf("%w: %s raised %v", port.ErrNotADocument, what, raised)
	}
}

// mapReadError says which of the two ways a document could not be read, in the words
// the core knows them by.
func mapReadError(err error) error {
	switch {
	case errors.Is(err, ErrEncrypted):
		return fmt.Errorf("%w: %w", port.ErrEncrypted, err)
	case errors.Is(err, ErrNotPDF):
		return fmt.Errorf("%w: %w", port.ErrNotADocument, err)
	}
	return err
}
