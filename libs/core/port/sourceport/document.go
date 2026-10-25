package sourceport

import (
	"context"
	"errors"
	"image"
	"io"
)

// PageRenderer draws the pages of a document as pictures, for a screen to
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
	RenderPage(index, dpi int) (image.Image, error)
	Close()
}

// ErrNotADocument is bytes that are not a document the renderer draws.
var ErrNotADocument = errors.New("not a document this reads")
