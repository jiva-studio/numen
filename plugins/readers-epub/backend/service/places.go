package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/api"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
)

// placedBook is the book a place was last asked of, kept while its file is the same bytes: the tab
// asks for the CFI of each page it turns to.
type placedBook struct {
	key   string
	size  int64
	mtime int64
	book  *epub.Book
}

// readPlacedBook reads a book of the vault the request works, or returns the one read last while
// its file is unchanged.
func (s *BookServer) readPlacedBook(ctx context.Context, path string) (*epub.Book, error) {
	if path == "" {
		return nil, server.NewAPIError(server.CodeInvalidArgument, "path required")
	}
	vaultID := getRequestVault(ctx)
	files := getVault(ctx, s.files)
	info, err := files.StatFile(ctx, path)
	if err != nil {
		return nil, readVaultError(path, err)
	}
	key := string(vaultID) + "\x00" + path
	s.mu.Lock()
	last := s.placed
	s.mu.Unlock()
	if last.book != nil && last.key == key && last.size == info.Size && last.mtime == info.ModTime.UnixNano() {
		return last.book, nil
	}
	raw, info, err := files.Read(ctx, path)
	if err != nil {
		return nil, readVaultError(path, err)
	}
	book, err := epub.Read(raw)
	if err != nil {
		return nil, newReadError(err)
	}
	s.mu.Lock()
	s.placed = placedBook{key: key, size: info.Size, mtime: info.ModTime.UnixNano(), book: book}
	s.mu.Unlock()
	return book, nil
}

// GetCfi returns the EPUB CFI of the character at an offset of a book's text.
func (s *BookServer) GetCfi(ctx context.Context, req api.GetCfiRequest) (api.GetCfiResponse, error) {
	book, err := s.readPlacedBook(ctx, req.Path)
	if err != nil {
		return api.GetCfiResponse{}, err
	}
	place, ok := book.GetCFI(req.Offset)
	if !ok {
		return api.GetCfiResponse{}, server.NewAPIError(server.CodeNotFound, fmt.Sprintf("%s has no text to name a place in", req.Path))
	}
	return api.GetCfiResponse{Cfi: place.String()}, nil
}

// FindOffset returns the offset of a book's text an EPUB CFI names.
func (s *BookServer) FindOffset(ctx context.Context, req api.FindOffsetRequest) (api.FindOffsetResponse, error) {
	place, err := cfi.Parse(req.Cfi)
	if err != nil {
		return api.FindOffsetResponse{}, server.NewAPIError(server.CodeInvalidArgument, err.Error())
	}
	book, err := s.readPlacedBook(ctx, req.Path)
	if err != nil {
		return api.FindOffsetResponse{}, err
	}
	offset, ok := book.FindOffset(place)
	if !ok {
		return api.FindOffsetResponse{}, server.NewAPIError(server.CodeNotFound, fmt.Sprintf("%s names no document of %s", req.Cfi, req.Path))
	}
	return api.FindOffsetResponse{Offset: offset}, nil
}
