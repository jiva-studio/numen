package sourceport

import (
	"context"
	"errors"
	"image"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// TextExtractor takes the text out of a file whose text is laid out on printed
// pages, and says where on a page each of its words sits.
//
// The format needs a library to read, and the library holds workers for as long
// as the application does. What the core knows of it is here.
type TextExtractor interface {
	// Read is the whole of what a document says, the parts it names, and where
	// each of its pages begins in that text.
	Read(ctx context.Context, raw []byte) (domain.TextLayer, error)

	// GetHighlights is where the words of the pages named sit, as fractions of the
	// page, one box a word. Starts is where each page begins, as Read answered,
	// and pages are the ones wanted, by index.
	GetHighlights(ctx context.Context, raw []byte, starts []int, pages []int) ([]domain.HighlightBox, error)

	// ReadBook extracts text, sections, and print page mapping from a book.
	// Invalid file contents return ErrNotADocument, and encrypted files return ErrEncrypted.
	ReadBook(ctx context.Context, raw []byte) (domain.BookText, error)
}

// PageRenderer draws the pages of such a document as pictures, for a screen to
// show or a model to read.
type PageRenderer interface {
	// OpenDocument holds a document open so its pages can be drawn. It holds a worker
	// until it is closed, and there are as many workers as this machine has
	// cores.
	//
	// The document is read from the file as its pages are wanted. OpenDocument owns the
	// file from the call on and closes it when the document is closed, and at
	// once when the document does not open.
	OpenDocument(ctx context.Context, file io.ReadSeekCloser) (OpenDocument, error)
}

// OpenDocument represents an open document, drawn a page at a time. Closing it releases
// the underlying worker.
type OpenDocument interface {
	GetPageCount() int
	GetPageSize(index int) (width, height float64, err error)
	RenderPage(index, dpi int) (image.Image, error)
	Close()
}

// DocumentRenderer is an alias for PageRenderer.
type DocumentRenderer = PageRenderer

// Errors returned when reading documents or books fails.
var (
	// ErrNotADocument is bytes that are not the format at all.
	ErrNotADocument = errors.New("not a document this reads")
	// ErrEncrypted is a document whose text is locked.
	ErrEncrypted = errors.New("the document is locked")
	// ErrNoBook is a path that holds no book.
	ErrNoBook = errors.New("there is no such book")
	// ErrNoBookEntry is an entry the archive of a book does not hold.
	ErrNoBookEntry = errors.New("the book has no such entry")
	// ErrBookChanged is a book whose file is no longer the bytes an ask named.
	ErrBookChanged = errors.New("the book changed since it was asked for")
)
