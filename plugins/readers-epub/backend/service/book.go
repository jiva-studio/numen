package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
	"github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1/epubv1connect"
)

const mostListed = 1000

// readBookFile reads a book of the vault the request works.
func (s *BookServer) readBookFile(ctx context.Context, header http.Header, path string) ([]byte, error) {
	cleanPath, err := s.vaults.getFilePath(ctx, header, path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}
	return raw, nil
}

// BookServer implements epubv1connect.BookServiceHandler. Each call works the vault its Numen-Vault
// header names.
type BookServer struct {
	epubv1connect.UnimplementedBookServiceHandler

	vaults vaultRoots

	// books caches the books entries are read from, and closes them when the server closes.
	books *bookCache
}

// NewBookServer returns a BookServer that finds the vault of a request through files, and works the
// vault at defaultRoot for a request naming none.
func NewBookServer(files host.Vault, defaultRoot func() string) *BookServer {
	return &BookServer{vaults: vaultRoots{files: files, defaultRoot: defaultRoot}, books: newBookCache()}
}

// GetBook parses an EPUB file and returns metadata, documents, and parts.
func (s *BookServer) GetBook(
	ctx context.Context,
	req *connect.Request[epubv1.GetBookRequest],
) (*connect.Response[epubv1.GetBookResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	raw, err := s.readBookFile(ctx, req.Header(), path)
	if err != nil {
		return nil, err
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse epub %s: %w", path, err))
	}

	resp := &epubv1.GetBookResponse{
		Title:     book.Title,
		PageCount: int32(book.PageCount()),
		TextBytes: int32(len(book.Text)),
		PageBytes: int32(book.PageBytes()),
	}

	docCount := min(len(book.Documents), mostListed)
	for _, doc := range book.Documents[:docCount] {
		resp.Documents = append(resp.Documents, &epubv1.SpineDocument{
			Path:   doc.Path,
			Offset: int32(doc.Offset),
			Length: int32(doc.Length),
		})
	}

	partCount := min(len(book.Parts), mostListed)
	for _, part := range book.Parts[:partCount] {
		resp.Parts = append(resp.Parts, &epubv1.BookPart{
			Title:  part.Title,
			Offset: int32(part.Offset),
			Level:  int32(part.Level),
		})
	}

	pageCount := min(len(book.Pages), mostListed)
	for _, page := range book.Pages[:pageCount] {
		resp.PrintedPages = append(resp.PrintedPages, &epubv1.PrintedPage{
			Label:  page.Label,
			Offset: int32(page.Offset),
		})
	}

	for _, anchor := range book.Anchors() {
		resp.Anchors = append(resp.Anchors, &epubv1.Anchor{Offset: int32(anchor.Offset), Cfi: anchor.CFI})
	}

	return connect.NewResponse(resp), nil
}

// ReadBookMarkup returns rendered HTML markup for an EPUB document spine.
func (s *BookServer) ReadBookMarkup(
	ctx context.Context,
	req *connect.Request[epubv1.ReadBookMarkupRequest],
) (*connect.Response[epubv1.ReadBookMarkupResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	raw, err := s.readBookFile(ctx, req.Header(), path)
	if err != nil {
		return nil, err
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse epub %s: %w", path, err))
	}

	rendered, err := book.Markup(req.Msg.GetDocument())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read document markup %s: %w", req.Msg.GetDocument(), err))
	}

	return connect.NewResponse(&epubv1.ReadBookMarkupResponse{
		Markup: rendered.HTML(),
	}), nil
}

// ReadBookEntry returns the bytes of one entry of an EPUB's archive while the
// file is still the bytes the baseline names.
func (s *BookServer) ReadBookEntry(
	ctx context.Context,
	req *connect.Request[epubv1.ReadBookEntryRequest],
) (*connect.Response[epubv1.ReadBookEntryResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	if req.Msg.GetEntry() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("entry required"))
	}
	cleanPath, err := s.vaults.getFilePath(ctx, req.Header(), path)
	if err != nil {
		return nil, err
	}
	book, err := s.books.acquire(cleanPath)
	if err != nil {
		if errors.Is(err, errCacheClosed) {
			return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("read book %s: %w", path, err))
		}
		if errors.Is(err, errReadFile) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
		}
		return nil, newReadError(err)
	}
	defer s.books.release(book)
	if baseline := req.Msg.GetBaseline(); baseline != nil && !isFingerprintOf(baseline, book.size, book.mtime) {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("read book %s: the file changed", path))
	}
	body, err := book.archive.Entry(req.Msg.GetEntry())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&epubv1.ReadBookEntryResponse{Body: body}), nil
}

// isFingerprintOf is whether a file of the size and the stamp given is the bytes the
// fingerprint names. A stamp of zero names no time.
func isFingerprintOf(named *numenv1.Fingerprint, size, mtime int64) bool {
	if named.GetSize() != size {
		return false
	}
	return named.GetMtime() == 0 || named.GetMtime() == mtime
}

// locked book is a failed precondition, and bytes that are no book are an invalid
// argument.
func newReadError(err error) error {
	if errors.Is(err, epub.ErrEncrypted) {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read epub: %w", err))
	}
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse epub: %w", err))
}

// Close closes the cached books.
func (s *BookServer) Close() {
	s.books.Close()
}
