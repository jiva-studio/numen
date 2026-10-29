package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// errChanged is what reading an entry fails with where the file is no longer the bytes the caller
// read.
var errChanged = errors.New("the file changed")

const mostListed = 1000

// readBookFile reads a book of the vault the request works.
func (s *BookServer) readBookFile(ctx context.Context, path string) ([]byte, error) {
	raw, _, err := getVault(ctx, s.files).Read(ctx, path)
	if err != nil {
		return nil, readVaultError(path, err)
	}
	return raw, nil
}

// BookServer implements the methods of the numen.epub service. Each call works the vault its context
// names with withRequestVault.
type BookServer struct {
	files host.Vault

	// books caches the books entries are read from, and closes them when the server closes.
	books *bookCache

	mu     sync.Mutex
	placed placedBook
}

// NewBookServer returns a BookServer that reads the vault of a request through files.
func NewBookServer(files host.Vault) *BookServer {
	return &BookServer{files: files, books: newBookCache()}
}

// GetBook parses an EPUB file and returns metadata, documents, and parts.
func (s *BookServer) GetBook(ctx context.Context, req api.GetBookRequest) (api.GetBookResponse, error) {
	if req.Path == "" {
		return api.GetBookResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	raw, err := s.readBookFile(ctx, req.Path)
	if err != nil {
		return api.GetBookResponse{}, err
	}

	book, err := epub.Read(raw)
	if err != nil {
		return api.GetBookResponse{}, server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("parse epub %s: %v", req.Path, err))
	}

	res := api.GetBookResponse{
		Title:        book.Title,
		PageCount:    book.PageCount(),
		TextBytes:    len(book.Text),
		PageBytes:    book.PageBytes(),
		Documents:    []api.SpineDocument{},
		Parts:        []api.BookPart{},
		PrintedPages: []api.PrintedPage{},
	}

	docCount := min(len(book.Documents), mostListed)
	for _, doc := range book.Documents[:docCount] {
		res.Documents = append(res.Documents, api.SpineDocument{Path: doc.Path, Offset: doc.Offset, Length: doc.Length})
	}

	partCount := min(len(book.Parts), mostListed)
	for _, part := range book.Parts[:partCount] {
		res.Parts = append(res.Parts, api.BookPart{Title: part.Title, Offset: part.Offset, Level: part.Level})
	}

	pageCount := min(len(book.Pages), mostListed)
	for _, page := range book.Pages[:pageCount] {
		res.PrintedPages = append(res.PrintedPages, api.PrintedPage{Label: page.Label, Offset: page.Offset})
	}

	return res, nil
}

// ReadBookMarkup returns rendered HTML markup for an EPUB document spine.
func (s *BookServer) ReadBookMarkup(ctx context.Context, req api.ReadBookMarkupRequest) (api.ReadBookMarkupResponse, error) {
	if req.Path == "" {
		return api.ReadBookMarkupResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	raw, err := s.readBookFile(ctx, req.Path)
	if err != nil {
		return api.ReadBookMarkupResponse{}, err
	}

	book, err := epub.Read(raw)
	if err != nil {
		return api.ReadBookMarkupResponse{}, server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("parse epub %s: %v", req.Path, err))
	}

	rendered, err := book.Markup(req.Document)
	if err != nil {
		return api.ReadBookMarkupResponse{}, server.NewAPIError(server.CodeNotFound, fmt.Sprintf("read document markup %s: %v", req.Document, err))
	}

	return api.ReadBookMarkupResponse{Markup: rendered.HTML()}, nil
}

// ReadBookEntry returns the bytes of one entry of an EPUB's archive. A baseline that does not name
// the bytes of the file fails with errChanged.
func (s *BookServer) ReadBookEntry(ctx context.Context, req api.ReadBookEntryRequest) (api.ReadBookEntryResponse, error) {
	if req.Path == "" {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	if req.Entry == "" {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeInvalidArgument, "entry required")
	}
	files := getVault(ctx, s.files)
	book, err := s.books.acquire(ctx, files, getRequestVault(ctx), req.Path)
	if err != nil {
		if errors.Is(err, errCacheClosed) {
			return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeUnavailable, fmt.Sprintf("read book %s: %v", req.Path, err))
		}
		if errors.Is(err, errReadFile) {
			return api.ReadBookEntryResponse{}, readVaultError(req.Path, err)
		}
		return api.ReadBookEntryResponse{}, newReadError(err)
	}
	defer s.books.release(book)
	if req.Baseline != nil && !isStatOf(*req.Baseline, book.size, book.mtime) {
		return api.ReadBookEntryResponse{}, fmt.Errorf("read book %s: %w", req.Path, errChanged)
	}
	body, err := book.archive.Entry(req.Entry)
	if err != nil {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeNotFound, err.Error())
	}
	return api.ReadBookEntryResponse{Body: body}, nil
}

// isStatOf is whether a file of the size and the stamp given is the bytes the
// file stat names. A stamp of zero names no time.
func isStatOf(named api.FileStat, size, mtime int64) bool {
	if named.Size != size {
		return false
	}
	return named.Mtime == 0 || named.Mtime == mtime
}

// locked book is a failed precondition, and bytes that are no book are an invalid
// argument.
func newReadError(err error) error {
	if errors.Is(err, epub.ErrEncrypted) {
		return server.NewAPIError(server.CodeFailedPrecondition, fmt.Sprintf("read epub: %v", err))
	}
	return server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("parse epub: %v", err))
}

// Close closes the cached books.
func (s *BookServer) Close() {
	s.books.Close()
}
