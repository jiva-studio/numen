package editor

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/epub"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/document"
)

// A book that reflows reaches the window as words: the markup of one document
// at a time, asked for over the schema, and beside it the pictures the archive
// carries, answered at an address as the bytes they are.

// mostListed is how many documents, parts and printed pages of one book cross.
// A book names as many as it likes.
const mostListed = 10_000

// GetBook answers what a book that reflows is: what it is called, the documents
// it is read in, and what it names inside them.
//
// A window lays a chapter out itself, so it takes the whole of the book's shape
// here and then asks for one document at a time.
func (a *API) GetBook(
	ctx context.Context,
	r *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	if a.Documents.GetBook == nil {
		return nil, errNoDocuments
	}
	showing, err := a.getShownVault()
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	ctx, cancel := context.WithTimeout(ctx, a.Viewer.patience)
	defer cancel()

	read, err := a.Documents.GetBook.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}

	out := &v1.GetBookResponse{
		Title:       read.Title,
		PageCount:   int32(read.PageCount()),
		TextBytes:   int32(len(read.Text)),
		PageBytes:   int32(read.PageBytes()),
		Fingerprint: newWireFingerprint(read.Fingerprint),
	}
	for _, doc := range read.Documents[:min(len(read.Documents), mostListed)] {
		out.Documents = append(out.Documents, &v1.SpineDocument{
			Path:   doc.Path,
			Offset: int32(doc.Offset),
			Length: int32(doc.Length),
		})
	}
	for _, part := range read.Parts[:min(len(read.Parts), mostListed)] {
		out.Parts = append(out.Parts, &v1.BookPart{
			Title:  part.Title,
			Offset: int32(part.Offset),
			Level:  int32(part.Level),
		})
	}
	for _, page := range read.Pages[:min(len(read.Pages), mostListed)] {
		out.PrintedPages = append(out.PrintedPages, &v1.PrintedPage{Label: page.Label, Offset: int32(page.Offset)})
	}
	return connect.NewResponse(out), nil
}

// ReadBookMarkup answers with one document of a book, as the markup a window
// sets a page from.
//
// The ask names the bytes it was given out for, and is answered while the file
// is still those bytes: one answer is one reading of one file.
func (a *API) ReadBookMarkup(
	ctx context.Context,
	r *connect.Request[v1.ReadBookMarkupRequest],
) (*connect.Response[v1.ReadBookMarkupResponse], error) {
	if a.Documents.GetBookMarkup == nil {
		return nil, errNoDocuments
	}
	showing, err := a.getShownVault()
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	ctx, cancel := context.WithTimeout(ctx, a.Viewer.patience)
	defer cancel()

	var baseline *document.Fingerprint
	if named := r.Msg.GetBaseline(); named != nil {
		baseline = &document.Fingerprint{Size: named.GetSize(), Mtime: named.GetMtime()}
	}
	markup, err := a.Documents.GetBookMarkup.Execute(ctx, showing, r.Msg.GetPath(), baseline, r.Msg.GetDocument())
	if err != nil {
		return nil, connect.NewError(getDrawCode(err), err)
	}
	return connect.NewResponse(&v1.ReadBookMarkupResponse{Markup: markup}), nil
}

// Entry answers with the bytes of one entry of a book's archive, which is how a
// picture in a book is drawn.
//
// The bytes are somebody else's: a book in a synced vault was put there by
// whoever synced it, and this route serves it from the window's own origin. So
// the type is settled by reading the bytes, and an entry that is not one of the
// pictures a book is drawn from is not served at all.
func (a *API) Entry(w http.ResponseWriter, r *http.Request, path, entry string) {
	named, err := parseFingerprint(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.Documents.GetBookPicture == nil {
		http.Error(w, errNoDocuments.Error(), http.StatusNotImplemented)
		return
	}
	showing, err := a.getShownVault()
	if err != nil {
		refuse(w, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	picture, err := a.Documents.GetBookPicture.Execute(ctx, showing, path, named, entry)
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
	return errors.Is(err, epub.ErrNotArchive) ||
		errors.Is(err, epub.ErrNoContainer) ||
		errors.Is(err, epub.ErrNoPackage) ||
		errors.Is(err, epub.ErrEncrypted) ||
		errors.Is(err, document.ErrNotAPicture)
}
