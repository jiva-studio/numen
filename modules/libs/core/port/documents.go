package port

import (
	"context"
	"errors"
	"image"

	"github.com/jiva-studio/numen/modules/libs/core/cutting"
	"github.com/jiva-studio/numen/modules/libs/core/lit"
)

// Documents reads a file whose text is laid out on printed pages: what it
// says, where a run of that text sits on a page, and a page as a picture.
//
// The format needs a library to read, and the library holds workers for as long
// as the application does. What the core knows of it is here.
type Documents interface {
	// Read is the whole of what a document says, the parts it names, and where
	// each of its pages begins in that text.
	Read(ctx context.Context, raw []byte) (Reading, error)

	// Lit is where the words of the pages named sit, as fractions of the page,
	// one box a word. Starts is where each page begins, as Read answered, and
	// pages are the ones wanted, by index.
	Lit(ctx context.Context, raw []byte, starts []int, pages []int) ([]lit.Box, error)

	// Draw holds a document open so its pages can be drawn. It holds a worker
	// until it is closed, and there are as many workers as this machine has
	// cores.
	Draw(ctx context.Context, raw []byte) (OpenDocument, error)
}

// A Reading is one document, read.
type Reading struct {
	// Text is every page's text in the order the document is paginated, as one
	// stream. Every offset below is an offset into it.
	Text string
	// Parts are the names the document gives divisions of itself.
	Parts []cutting.PartStart
	// Pages is where each page begins.
	Pages []int
}

// OpenDocument is a document held open, drawn a page at a time. Closing it gives back
// the worker it holds.
type OpenDocument interface {
	Pages() int
	Size(index int) (wide, high float64, err error)
	Image(index, dpi int) (image.Image, error)
	Close()
}

// A document that cannot be read is one of these, and a caller says which of
// the two it met.
var (
	// ErrNotADocument is bytes that are not the format at all.
	ErrNotADocument = errors.New("not a document this reads")
	// ErrEncrypted is a document whose text is locked.
	ErrEncrypted = errors.New("the document is locked")
)
