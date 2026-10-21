package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/vault"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/readers.epub/backend/epub"
)

const mostListed = 1000

func (s *BookServer) readBookFile(path string) ([]byte, error) {
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleanPath)
}

// BookServer implements numenv1connect.BookServiceHandler.
type BookServer struct {
	numenv1connect.UnimplementedBookServiceHandler

	// vaultRoot is the vault the paths of requests are held to.
	vaultRoot func() string

	// books caches the books entries are read from, and closes them when the server closes.
	books *bookCache
}

// NewBookServer returns a BookServer that restricts paths to the vault NUMEN_VAULT_PATH names.
func NewBookServer() *BookServer {
	return &BookServer{vaultRoot: func() string { return os.Getenv("NUMEN_VAULT_PATH") }, books: newBookCache()}
}

// NewBookServerIn returns a BookServer that restricts paths to the vault at vaultRoot.
func NewBookServerIn(vaultRoot string) *BookServer {
	return &BookServer{vaultRoot: func() string { return vaultRoot }, books: newBookCache()}
}

// GetBook parses an EPUB file and returns metadata, documents, and parts.
func (s *BookServer) GetBook(
	_ context.Context,
	req *connect.Request[numenv1.GetBookRequest],
) (*connect.Response[numenv1.GetBookResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	raw, err := s.readBookFile(path)
	if err != nil {
		if errors.Is(err, vault.ErrOutsideVault) || errors.Is(err, vault.ErrPathTraversal) || errors.Is(err, vault.ErrEmptyVault) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read book %s: %w", path, err))
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse epub %s: %w", path, err))
	}

	resp := &numenv1.GetBookResponse{
		Title:     book.Title,
		PageCount: int32(book.PageCount()),
		TextBytes: int32(len(book.Text)),
		PageBytes: int32(book.PageBytes()),
	}

	docCount := min(len(book.Documents), mostListed)
	for _, doc := range book.Documents[:docCount] {
		resp.Documents = append(resp.Documents, &numenv1.SpineDocument{
			Path:   doc.Path,
			Offset: int32(doc.Offset),
			Length: int32(doc.Length),
		})
	}

	partCount := min(len(book.Parts), mostListed)
	for _, part := range book.Parts[:partCount] {
		resp.Parts = append(resp.Parts, &numenv1.BookPart{
			Title:  part.Title,
			Offset: int32(part.Offset),
			Level:  int32(part.Level),
		})
	}

	pageCount := min(len(book.Pages), mostListed)
	for _, page := range book.Pages[:pageCount] {
		resp.PrintedPages = append(resp.PrintedPages, &numenv1.PrintedPage{
			Label:  page.Label,
			Offset: int32(page.Offset),
		})
	}

	return connect.NewResponse(resp), nil
}

// ReadBookMarkup returns rendered HTML markup for an EPUB document spine.
func (s *BookServer) ReadBookMarkup(
	_ context.Context,
	req *connect.Request[numenv1.ReadBookMarkupRequest],
) (*connect.Response[numenv1.ReadBookMarkupResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	raw, err := s.readBookFile(path)
	if err != nil {
		if errors.Is(err, vault.ErrOutsideVault) || errors.Is(err, vault.ErrPathTraversal) || errors.Is(err, vault.ErrEmptyVault) {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read book %s: %w", path, err))
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}

	book, err := epub.Read(raw)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse epub %s: %w", path, err))
	}

	drawn, err := book.Markup(req.Msg.GetDocument())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read document markup %s: %w", req.Msg.GetDocument(), err))
	}

	return connect.NewResponse(&numenv1.ReadBookMarkupResponse{
		Markup: drawn.HTML(),
	}), nil
}

// ExtractBookText returns the text of the EPUB in the bytes given, with the parts
// and the printed pages it names.
func (s *BookServer) ExtractBookText(
	_ context.Context,
	req *connect.Request[numenv1.ExtractBookTextRequest],
) (*connect.Response[numenv1.ExtractBookTextResponse], error) {
	if len(req.Msg.GetData()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("data required"))
	}
	book, err := epub.Read(req.Msg.GetData())
	if err != nil {
		return nil, newReadError(err)
	}
	resp := &numenv1.ExtractBookTextResponse{Text: book.Text}
	for _, part := range book.Parts {
		resp.Parts = append(resp.Parts, &numenv1.BookPart{
			Title:  part.Title,
			Offset: int32(part.Offset),
			Level:  int32(part.Level),
		})
	}
	for _, page := range book.Pages {
		resp.PrintedPages = append(resp.PrintedPages, &numenv1.PrintedPage{
			Label:  page.Label,
			Offset: int32(page.Offset),
		})
	}
	return connect.NewResponse(resp), nil
}

// ReadBookEntry returns the bytes of one entry of an EPUB's archive while the
// file is still the bytes the baseline names.
func (s *BookServer) ReadBookEntry(
	_ context.Context,
	req *connect.Request[numenv1.ReadBookEntryRequest],
) (*connect.Response[numenv1.ReadBookEntryResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("path required"))
	}
	if req.Msg.GetEntry() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("entry required"))
	}
	cleanPath, err := vault.ValidatePathIn(s.vaultRoot(), path)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read book %s: %w", path, err))
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
	return connect.NewResponse(&numenv1.ReadBookEntryResponse{Body: body}), nil
}

// isFingerprintOf is whether a file of the size and the stamp given is the bytes the
// fingerprint names. A stamp of zero names no time.
func isFingerprintOf(named *numenv1.Fingerprint, size, mtime int64) bool {
	if named.GetSize() != size {
		return false
	}
	return named.GetMtime() == 0 || named.GetMtime() == mtime
}

// newReadError is the status a book that could not be read is refused under: a
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
