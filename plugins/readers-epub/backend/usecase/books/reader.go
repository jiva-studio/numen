// Package books reads EPUB books of a vault for the md.numen.readers.epub service: their structure, the
// markup of their documents, the entries of their archives and the places of their text.
package books

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/domain/epub"
)

// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="mostListed caps the maximum number of items returned in a book summary"
const mostListed = 1000

func (s *Reader) readBookFile(ctx context.Context, path string) ([]byte, error) {
	raw, _, err := s.vaults(getVault(ctx)).Read(ctx, path)
	if err != nil {
		return nil, readVaultError(path, err)
	}
	return raw, nil
}

// Reader implements the book methods of the md.numen.readers.epub service. Each call works the vault its
// context names with WithVault.
type Reader struct {
	vaults func(domain.VaultID) host.FileReader

	books *bookCache

	mu     sync.Mutex
	placed placedBook
}

// NewReader returns a Reader that reads the vault of a request through vaults.
func NewReader(vaults func(domain.VaultID) host.FileReader) *Reader {
	return &Reader{
		vaults: vaults,
		books:  newBookCache(),
		mu:     sync.Mutex{},
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="placedBook starts with nil book so size and mtime are ignored until populated"
		placed: placedBook{key: "", size: 0, mtime: 0, book: nil},
	}
}

// GetBook parses an EPUB file and returns metadata, documents, and parts.
func (s *Reader) GetBook(ctx context.Context, req api.GetBookRequest) (api.GetBookResponse, error) {
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
		Stat:         nil,
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
func (s *Reader) ReadBookMarkup(ctx context.Context, req api.ReadBookMarkupRequest) (api.ReadBookMarkupResponse, error) {
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
// the bytes of the file fails with a failed-precondition error.
func (s *Reader) ReadBookEntry(ctx context.Context, req api.ReadBookEntryRequest) (api.ReadBookEntryResponse, error) {
	if req.Path == "" {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	if req.Entry == "" {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeInvalidArgument, "entry required")
	}
	files := s.vaults(getVault(ctx))
	book, err := s.books.acquire(ctx, files, getVault(ctx), req.Path)
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
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeFailedPrecondition, fmt.Sprintf("read book %s: the file changed", req.Path))
	}
	body, err := book.archive.Entry(req.Entry)
	if err != nil {
		return api.ReadBookEntryResponse{}, server.NewAPIError(server.CodeNotFound, err.Error())
	}
	return api.ReadBookEntryResponse{Body: body}, nil
}

func isStatOf(named api.FileStat, size, mtime int64) bool {
	if named.Size != size {
		return false
	}
	return named.Mtime == 0 || named.Mtime == mtime
}

func newReadError(err error) error {
	if errors.Is(err, epub.ErrEncrypted) {
		return server.NewAPIError(server.CodeFailedPrecondition, fmt.Sprintf("read epub: %v", err))
	}
	return server.NewAPIError(server.CodeInvalidArgument, fmt.Sprintf("parse epub: %v", err))
}

// Close closes the cached books.
func (s *Reader) Close() {
	s.books.Close()
}
