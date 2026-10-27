package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
	epubv1 "github.com/jiva-studio/numen/plugins/readers-epub/gen/numen/epub/v1"
)

// placedBook is the book a place was last asked of, kept while its file is the same bytes: the tab
// asks for the CFI of each page it turns to.
type placedBook struct {
	path  string
	size  int64
	mtime int64
	book  *epub.Book
}

// readPlacedBook reads a book of the vault the request works, or returns the one read last while
// its file is unchanged.
func (s *BookServer) readPlacedBook(ctx context.Context, header http.Header, path string) (*epub.Book, error) {
	if path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("path required"))
	}
	cleanPath, err := s.vaults.getFilePath(ctx, header, path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}
	s.mu.Lock()
	last := s.placed
	s.mu.Unlock()
	if last.book != nil && last.path == cleanPath && last.size == info.Size() && last.mtime == info.ModTime().UnixNano() {
		return last.book, nil
	}
	raw, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("read book %s: %w", path, err))
	}
	book, err := epub.Read(raw)
	if err != nil {
		return nil, newReadError(err)
	}
	s.mu.Lock()
	s.placed = placedBook{path: cleanPath, size: info.Size(), mtime: info.ModTime().UnixNano(), book: book}
	s.mu.Unlock()
	return book, nil
}

// GetCfi returns the EPUB CFI of the character at an offset of a book's text.
func (s *BookServer) GetCfi(
	ctx context.Context,
	req *connect.Request[epubv1.GetCfiRequest],
) (*connect.Response[epubv1.GetCfiResponse], error) {
	book, err := s.readPlacedBook(ctx, req.Header(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	place, ok := book.GetCFI(int(req.Msg.GetOffset()))
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("%s has no text to name a place in", req.Msg.GetPath()))
	}
	return connect.NewResponse(&epubv1.GetCfiResponse{Cfi: place.String()}), nil
}

// FindOffset returns the offset of a book's text an EPUB CFI names.
func (s *BookServer) FindOffset(
	ctx context.Context,
	req *connect.Request[epubv1.FindOffsetRequest],
) (*connect.Response[epubv1.FindOffsetResponse], error) {
	place, err := cfi.Parse(req.Msg.GetCfi())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	book, err := s.readPlacedBook(ctx, req.Header(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	offset, ok := book.FindOffset(place)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("%s names no document of %s", req.Msg.GetCfi(), req.Msg.GetPath()))
	}
	return connect.NewResponse(&epubv1.FindOffsetResponse{Offset: int32(offset)}), nil
}
