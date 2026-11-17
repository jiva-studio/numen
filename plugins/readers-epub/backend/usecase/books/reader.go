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
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

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

// GetBook parses an EPUB file and returns its metadata, documents and parts.
func (s *Reader) GetBook(ctx context.Context, path string) (*epub.Book, error) {
	if path == "" {
		return nil, newError(KindInvalidArgument, "path required")
	}
	raw, err := s.readBookFile(ctx, path)
	if err != nil {
		return nil, err
	}
	book, err := epub.Read(raw)
	if err != nil {
		return nil, newError(KindInvalidArgument, fmt.Sprintf("parse epub %s: %v", path, err))
	}
	return book, nil
}

// ReadBookMarkup returns the rendered HTML markup of a document of an EPUB's spine.
func (s *Reader) ReadBookMarkup(ctx context.Context, path, document string) (string, error) {
	if path == "" {
		return "", newError(KindInvalidArgument, "path required")
	}
	raw, err := s.readBookFile(ctx, path)
	if err != nil {
		return "", err
	}

	book, err := epub.Read(raw)
	if err != nil {
		return "", newError(KindInvalidArgument, fmt.Sprintf("parse epub %s: %v", path, err))
	}

	rendered, err := book.Markup(document)
	if err != nil {
		return "", newError(KindNotFound, fmt.Sprintf("read document markup %s: %v", document, err))
	}

	return rendered.HTML(), nil
}

// ReadBookEntry returns the bytes of one entry of an EPUB's archive. A baseline that does not name
// the bytes of the file fails with KindFailedPrecondition.
func (s *Reader) ReadBookEntry(ctx context.Context, path, entry string, baseline *Baseline) ([]byte, error) {
	if path == "" {
		return nil, newError(KindInvalidArgument, "path required")
	}
	if entry == "" {
		return nil, newError(KindInvalidArgument, "entry required")
	}
	files := s.vaults(getVault(ctx))
	book, err := s.books.acquire(ctx, files, getVault(ctx), path)
	if err != nil {
		if errors.Is(err, errCacheClosed) {
			return nil, newError(KindUnavailable, fmt.Sprintf("read book %s: %v", path, err))
		}
		if errors.Is(err, errReadFile) {
			return nil, readVaultError(path, err)
		}
		return nil, newReadError(err)
	}
	defer s.books.release(book)
	if baseline != nil && !isStatOf(*baseline, book.size, book.mtime) {
		return nil, newError(KindFailedPrecondition, fmt.Sprintf("read book %s: the file changed", path))
	}
	body, err := book.archive.Entry(entry)
	if err != nil {
		return nil, newError(KindNotFound, err.Error())
	}
	return body, nil
}

func isStatOf(named Baseline, size, mtime int64) bool {
	if named.Size != size {
		return false
	}
	return named.Mtime == 0 || named.Mtime == mtime
}

func newReadError(err error) error {
	if errors.Is(err, epub.ErrEncrypted) {
		return newError(KindFailedPrecondition, fmt.Sprintf("read epub: %v", err))
	}
	return newError(KindInvalidArgument, fmt.Sprintf("parse epub: %v", err))
}

// Close closes the cached books.
func (s *Reader) Close() {
	s.books.Close()
}
