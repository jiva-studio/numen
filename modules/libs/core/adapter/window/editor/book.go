package editor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// bookClient is what a book plugin answers: the shape of a book, one document of
// it as markup, and the bytes of one entry of its archive.
type bookClient interface {
	GetBook(context.Context, *connect.Request[v1.GetBookRequest]) (*connect.Response[v1.GetBookResponse], error)
	ReadBookMarkup(
		context.Context, *connect.Request[v1.ReadBookMarkupRequest],
	) (*connect.Response[v1.ReadBookMarkupResponse], error)
	ReadEntry(ctx context.Context, baseline *v1.Fingerprint, entry string) ([]byte, error)
}

// bookHandler answers what a book that reflows is, and the pictures it carries.
type bookHandler struct {
	numenv1connect.UnimplementedBookServiceHandler

	state          *State
	viewer         *viewer
	openClient     func(context.Context) (any, func(), error)
	getBookPicture *document.GetBookPicture
}

// A book that reflows reaches the window as words: the markup of one document
// at a time, asked for over the schema, and beside it the pictures the archive
// carries, answered at an address as the bytes they are.

// GetBook answers what a book that reflows is: what it is called, the documents
// it is read in, and what it names inside them.
//
// A window lays a chapter out itself, so it takes the whole of the book's shape
// here and then asks for one document at a time.
func (h *bookHandler) GetBook(
	ctx context.Context,
	r *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	client, closeClient, err := h.getClient(ctx)
	if err != nil {
		return nil, convertPluginError(err)
	}
	defer closeClient()
	res, err := client.GetBook(ctx, r)
	if err != nil {
		return nil, convertPluginError(err)
	}
	return res, nil
}

// ReadBookMarkup answers with one document of a book, as the markup a window
// sets a page from.
//
// The ask names the bytes it was given out for, and is answered while the file
// is still those bytes: one answer is one reading of one file.
func (h *bookHandler) ReadBookMarkup(
	ctx context.Context,
	r *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	client, closeClient, err := h.getClient(ctx)
	if err != nil {
		return nil, convertPluginError(err)
	}
	defer closeClient()
	res, err := client.ReadBookMarkup(ctx, r)
	if err != nil {
		return nil, convertPluginError(err)
	}
	return res, nil
}

// Entry answers with the bytes of one entry of a book's archive, which is how a
// picture in a book is drawn.
//
// The bytes are somebody else's: a book in a synced vault was put there by
// whoever synced it, and this route serves it from the window's own origin. So
// the type is settled by reading the bytes, and an entry that is not one of the
// pictures a book is drawn from is not served at all.
func (h *bookHandler) Entry(w http.ResponseWriter, r *http.Request, path, entry string) {
	named, err := parseFingerprint(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.getBookPicture == nil {
		http.Error(w, errNoDocuments.Error(), http.StatusNotImplemented)
		return
	}
	showing, err := h.state.getShownVault()
	if err != nil {
		refuse(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), h.viewer.patience)
	defer cancel()

	picture, err := h.getBookPicture.Execute(ctx, showing, path, named, entry)
	if err != nil {
		refuse(w, err)
		return
	}

	w.Header().Set("Content-Type", picture.Type)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(picture.Body)))
	w.Header().Set("Cache-Control", immutable)
	_, _ = w.Write(picture.Body)
}

// ReadEntry answers with the bytes of one entry of the archive of the book at a
// fingerprint, read by the book plugin.
func (h *bookHandler) ReadEntry(ctx context.Context, mark document.Fingerprint, entry string) ([]byte, error) {
	client, closeClient, err := h.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("open the book plugin: %w", err)
	}
	defer closeClient()
	return client.ReadEntry(ctx, newWireFingerprint(mark), entry)
}

// isMisnamed is whether the bytes are not what they are called: a file that is no
// book, and an entry of one that is no picture. It is the one thing a person can
// act on.
func isMisnamed(err error) bool {
	return errors.Is(err, document.ErrNotADocument) ||
		errors.Is(err, document.ErrEncrypted) ||
		errors.Is(err, document.ErrNotAPicture)
}

// getClient opens the book plugin's client, with the close that lets it go. A
// build with no book plugin, or one whose client answers none of a book's asks,
// is errNoDocuments.
func (h *bookHandler) getClient(ctx context.Context) (bookClient, func(), error) {
	if h.openClient == nil {
		return nil, nil, errNoDocuments
	}
	raw, closer, err := h.openClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	if closer == nil {
		closer = func() {}
	}
	client, ok := raw.(bookClient)
	if !ok {
		closer()
		return nil, nil, fmt.Errorf("%w: the book plugin answers with %T", errNoDocuments, raw)
	}
	return client, closer, nil
}

// convertPluginError is what a failure to open the book plugin, or of an ask of
// it, is answered with: a code the failure carries is kept, the end of the
// caller's context is coded as such, and anything else is coded as a file that
// could not be reached.
func convertPluginError(err error) error {
	var coded *connect.Error
	switch {
	case errors.As(err, &coded):
		return err
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	default:
		return connect.NewError(getReachCode(err), err)
	}
}
