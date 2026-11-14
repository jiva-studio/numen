package books

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub/cfi"
)

// placedBook is the book a place was last requested of, retained while its file is the same bytes: the tab
// requests the CFI of each page it turns to.
type placedBook struct {
	key   string
	size  int64
	mtime int64
	book  *epub.Book
}

func (s *Reader) readPlacedBook(ctx context.Context, path string) (*epub.Book, error) {
	if path == "" {
		return nil, newError(KindInvalidArgument, "path required")
	}
	vaultID := getVault(ctx)
	files := s.vaults(getVault(ctx))
	info, err := files.StatFile(ctx, path)
	if err != nil {
		return nil, readVaultError(path, err)
	}
	key := string(vaultID) + "\x00" + path
	s.mu.Lock()
	last := s.placed
	s.mu.Unlock()
	// gomutants:disable-next-line EXPRESSION_REMOVE,CONDITIONALS_NEGATION,INVERT_LOGICAL,BRANCH_IF reason="single-book cache avoids re-parsing unchanged EPUB bytes"
	if last.book != nil && last.key == key && last.size == info.Size && last.mtime == info.ModTime.UnixNano() {
		return last.book, nil
	}
	raw, info, err := files.Read(ctx, path)
	// gomutants:disable-next-line BRANCH_IF reason="propagates vault read failure after StatFile succeeded"
	if err != nil {
		return nil, readVaultError(path, err)
	}
	book, err := epub.Read(raw)
	if err != nil {
		return nil, newReadError(err)
	}
	s.mu.Lock()
	// gomutants:disable-next-line STATEMENT_REMOVE reason="caches the parsed book for subsequent place lookups"
	s.placed = placedBook{key: key, size: info.Size, mtime: info.ModTime.UnixNano(), book: book}
	s.mu.Unlock()
	return book, nil
}

// GetCfi returns the EPUB CFI of the character at an offset of a book's text.
func (s *Reader) GetCfi(ctx context.Context, path string, offset int) (string, error) {
	book, err := s.readPlacedBook(ctx, path)
	if err != nil {
		return "", err
	}
	place, ok := book.GetCFI(offset)
	if !ok {
		return "", newError(KindNotFound, fmt.Sprintf("%s has no text to name a place in", path))
	}
	return place.String(), nil
}

// FindOffset returns the offset of a book's text an EPUB CFI names.
func (s *Reader) FindOffset(ctx context.Context, path, cfiText string) (int, error) {
	place, err := cfi.Parse(cfiText)
	if err != nil {
		return 0, newError(KindInvalidArgument, err.Error())
	}
	book, err := s.readPlacedBook(ctx, path)
	if err != nil {
		return 0, err
	}
	offset, ok := book.FindOffset(place)
	if !ok {
		return 0, newError(KindNotFound, fmt.Sprintf("%s names no document of %s", cfiText, path))
	}
	return offset, nil
}
