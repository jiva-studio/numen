package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/modules/plugins/reader-epub/backend/epub"
)

const mostListed = 1000

var (
	ErrEmptyBookPath = errors.New("path required")
	ErrPathTraversal = errors.New("path traversal outside allowed scope")
	ErrOutsideVault  = errors.New("path outside allowed vault root")
)

// ValidateVaultPath ensures an EPUB book path is within the allowed vault scope.
func ValidateVaultPath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyBookPath
	}
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return "", ErrPathTraversal
	}

	vaultRoot := os.Getenv("NUMEN_VAULT_PATH")
	if vaultRoot != "" {
		cleanVault := filepath.Clean(vaultRoot)
		if !filepath.IsAbs(cleanPath) {
			cleanPath = filepath.Join(cleanVault, cleanPath)
		}
		prefix := cleanVault
		if !strings.HasSuffix(prefix, string(filepath.Separator)) {
			prefix += string(filepath.Separator)
		}
		if cleanPath != cleanVault && !strings.HasPrefix(cleanPath, prefix) {
			return "", ErrOutsideVault
		}
	}
	return cleanPath, nil
}

func readBookFile(path string) ([]byte, error) {
	cleanPath, err := ValidateVaultPath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(cleanPath)
}

// BookServer implements numenv1connect.BookServiceHandler.
type BookServer struct {
	numenv1connect.UnimplementedBookServiceHandler
}

// NewBookServer returns an initialized BookServer.
func NewBookServer() *BookServer {
	return &BookServer{}
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
	raw, err := readBookFile(path)
	if err != nil {
		if errors.Is(err, ErrOutsideVault) || errors.Is(err, ErrPathTraversal) {
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
	raw, err := readBookFile(path)
	if err != nil {
		if errors.Is(err, ErrOutsideVault) || errors.Is(err, ErrPathTraversal) {
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
	cleanPath, err := ValidateVaultPath(path)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("read book %s: %w", path, err))
	}
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}
	if baseline := req.Msg.GetBaseline(); baseline != nil && !isFingerprintOf(baseline, info) {
		return nil, connect.NewError(connect.CodeAborted, fmt.Errorf("read book %s: the file changed", path))
	}
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}
	book, err := epub.Read(raw)
	if err != nil {
		return nil, newReadError(err)
	}
	body, err := book.Entry(req.Msg.GetEntry())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&numenv1.ReadBookEntryResponse{Body: body}), nil
}

// isFingerprintOf is whether the file is the bytes the fingerprint names. A
// stamp of zero names no time.
func isFingerprintOf(named *numenv1.Fingerprint, info os.FileInfo) bool {
	if named.GetSize() != info.Size() {
		return false
	}
	return named.GetMtime() == 0 || named.GetMtime() == info.ModTime().UnixNano()
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
