package editor

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/epub"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// A book that reflows reaches the window as words: the markup of one document
// at a time, and beside it the pictures the archive carries. So a chapter is a
// handler, and what it answers with is the markup a page is drawn from.

// mostListed is how many documents, parts and printed pages of one book cross.
// A book names as many as it likes.
const mostListed = 10_000

// errNotAPicture is an entry of an archive that is not one of the pictures a
// book is drawn from.
var errNotAPicture = errors.New("this entry is not a picture")

// pictured are the types an entry may be served as, and what settles the type is
// the bytes. What the manifest calls a file is the book's own claim about it,
// and a book is somebody else's: an entry served as markup hands the window's
// origin to whoever wrote it.
//
// SVG is not among them. It is a document — script, style, and addresses of its
// own, which resolve against this route and reach no entry of the archive — and
// nothing about reading it out of an archive makes it a picture.
var pictured = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

// GetBook answers what a book that reflows is: what it is called, the documents
// it is read in, and what it names inside them.
//
// A window lays a chapter out itself, so it takes the whole of the book's shape
// here and then asks for one document at a time.
func (a *API) GetBook(
	ctx context.Context,
	r *connect.Request[v1.GetBookRequest],
) (*connect.Response[v1.GetBookResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, a.Viewer.patience)
	defer cancel()

	reader, print, err := a.stat(ctx, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}
	read, err := a.book(ctx, reader, print)
	if err != nil {
		return nil, connect.NewError(refusedDrawing(err), err)
	}

	out := &v1.GetBookResponse{
		Title:       read.Title,
		PageCount:   int32(read.PageCount()),
		TextBytes:   int32(len(read.Text)),
		PageBytes:   int32(read.PageBytes()),
		Fingerprint: &v1.Fingerprint{Path: print.path, Size: print.size, Mtime: print.mtime},
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

// Markup answers with one document of a book, as the markup a window draws a
// page from.
//
// The address names the bytes it was read from, so it is answered only while the
// file at that path is still those bytes and what it answers with never changes.
func (a *API) Markup(w http.ResponseWriter, r *http.Request, path, document string) {
	read, ok := a.reading(w, r, path)
	if !ok {
		return
	}
	drawn, err := read.Markup(document)
	if err != nil {
		refuse(w, err)
		return
	}
	body := drawn.HTML()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", immutable)
	_, _ = w.Write([]byte(body))
}

// Entry answers with the bytes of one entry of a book's archive, which is how a
// picture in a book is drawn.
//
// The bytes are somebody else's: a book in a synced vault was put there by
// whoever synced it, and this route serves it from the window's own origin. So
// the type is settled by reading the bytes, and an entry that is not one of the
// pictures a book is drawn from is not served at all.
func (a *API) Entry(w http.ResponseWriter, r *http.Request, path, entry string) {
	read, ok := a.reading(w, r, path)
	if !ok {
		return
	}
	body, err := read.Entry(entry)
	if err != nil {
		refuse(w, err)
		return
	}
	drawn := http.DetectContentType(body)
	if !pictured[drawn] {
		refuse(w, errNotAPicture)
		return
	}

	w.Header().Set("Content-Type", drawn)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Header().Set("Cache-Control", immutable)
	_, _ = w.Write(body)
}

// reading is the book an address is about, held open, and false where the
// caller has been told why it is not coming.
func (a *API) reading(w http.ResponseWriter, r *http.Request, path string) (*epub.Book, bool) {
	named, err := printed(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.Viewer.patience)
	defer cancel()

	reader, print, err := a.stat(ctx, path)
	if err != nil {
		refuse(w, err)
		return nil, false
	}
	if named.size != print.size || named.mtime != print.mtime {
		refuse(w, errChanged)
		return nil, false
	}
	read, err := a.book(ctx, reader, print)
	if err != nil {
		refuse(w, err)
		return nil, false
	}
	return read, true
}

// book hands over the book at a fingerprint, read where it is not held already.
//
// The bytes are read where the book is not open, and the read outlives the
// request that asked for it: a caller that ran out of patience is told the book
// is busy, and the book is there for the next ask.
func (a *API) book(
	ctx context.Context,
	reader port.VaultReader,
	print fingerprint,
) (*epub.Book, error) {
	return a.Viewer.read.Load().take(ctx, print, func() (*epub.Book, error) {
		raw, err := reader.Read(context.WithoutCancel(ctx), print.path)
		if err != nil {
			return nil, err
		}
		return epub.Read(raw)
	})
}

// misnamed is whether the bytes are not what they are called: a file that is no
// book, and an entry of one that is no picture. It is the one thing a person can
// act on.
func misnamed(err error) bool {
	return errors.Is(err, epub.ErrNotArchive) ||
		errors.Is(err, epub.ErrNoContainer) ||
		errors.Is(err, epub.ErrNoPackage) ||
		errors.Is(err, epub.ErrEncrypted) ||
		errors.Is(err, errNotAPicture)
}
