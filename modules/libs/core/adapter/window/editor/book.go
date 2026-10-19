package editor

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// bookHandler answers what a book that reflows is, and the pictures it carries.
type bookHandler struct {
	numenv1connect.UnimplementedBookServiceHandler

	state          *State
	viewer         *viewer
	getBook        *document.GetBook
	readBookMarkup *document.ReadBookMarkup
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
	if h.getBook == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
	}
	shape, err := h.getBook.Execute(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, convertBookError(err)
	}
	return connect.NewResponse(newWireBook(shape)), nil
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
	if h.readBookMarkup == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoDocuments)
	}
	var baseline *domain.Fingerprint
	if named := r.Msg.GetBaseline(); named != nil {
		baseline = &domain.Fingerprint{Path: named.GetPath(), Size: named.GetSize(), ModTime: instant(named.GetMtime())}
	}
	markup, err := h.readBookMarkup.Execute(ctx, r.Msg.GetPath(), r.Msg.GetDocument(), baseline)
	if err != nil {
		return nil, convertBookError(err)
	}
	return connect.NewResponse(&v1.ReadBookMarkupResponse{Markup: markup}), nil
}

// newWireBook is the shape of a book as the schema carries it.
func newWireBook(shape domain.BookShape) *v1.GetBookResponse {
	out := &v1.GetBookResponse{
		Title:       shape.Title,
		PageCount:   int32(shape.PageCount),
		TextBytes:   int32(shape.TextBytes),
		PageBytes:   int32(shape.PageBytes),
		Fingerprint: &v1.Fingerprint{Path: shape.Fingerprint.Path, Size: shape.Fingerprint.Size, Mtime: stamp(shape.Fingerprint.ModTime)},
	}
	for _, section := range shape.Sections {
		out.Documents = append(out.Documents, &v1.SpineDocument{
			Path: section.Path, Offset: int32(section.Offset), Length: int32(section.Length),
		})
	}
	for _, part := range shape.Parts {
		out.Parts = append(out.Parts, &v1.BookPart{
			Title: part.Title, Offset: int32(part.Offset), Level: int32(part.Level),
		})
	}
	for _, page := range shape.Pages {
		out.PrintedPages = append(out.PrintedPages, &v1.PrintedPage{Label: page.Label, Offset: int32(page.Offset)})
	}
	return out
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

// isMisnamed is whether the bytes are not what they are called: a file that is no
// book, and an entry of one that is no picture. It is the one thing a person can
// act on.
func isMisnamed(err error) bool {
	return errors.Is(err, document.ErrNotADocument) ||
		errors.Is(err, document.ErrEncrypted) ||
		errors.Is(err, document.ErrNotAPicture)
}

// convertBookError is the error an ask of a book is answered with. An absent
// book plugin is unimplemented, a book or an entry it lacks is not found, bytes
// that are no book are an invalid argument, a locked book a failed precondition
// and a file that changed aborted. A code the failure carries is kept, the end of
// the caller's context is coded as such, and anything else is coded as a file
// that could not be reached.
func convertBookError(err error) error {
	var coded *connect.Error
	switch {
	case errors.Is(err, port.ErrUnavailable):
		return connect.NewError(connect.CodeUnimplemented, err)
	case errors.Is(err, port.ErrNoBook), errors.Is(err, port.ErrNoBookEntry):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, port.ErrNotADocument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, port.ErrEncrypted):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, port.ErrBookChanged):
		return connect.NewError(connect.CodeAborted, err)
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
