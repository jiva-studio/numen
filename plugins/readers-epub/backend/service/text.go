package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/vault"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/epub"
)

// textKey is the key the plugin keeps the text of a book under.
const textKey = "text"

// The bounds of what one answer carries: a part holds at most partBytes of text, and a page of the
// text at most pageBytes, or one part where that is more.
const (
	partBytes = 64 << 10
	pageBytes = 1 << 20
)

// TextServer gives the host the text of the vault's EPUB books as plain text, each named section
// opened where it begins. The text and its offsets are the ones the reader sets the book with.
type TextServer struct {
	vaultRoot func() string
	texts     host.Texts

	mu   sync.Mutex
	last bookText
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the texts of the books of the vault at vaultRoot, reported to texts.
func NewTextServer(vaultRoot func() string, texts host.Texts) *TextServer {
	return &TextServer{vaultRoot: vaultRoot, texts: texts}
}

// bookText is one file's text as parts, and the hash of the bytes it was read from.
type bookText struct {
	hash  string
	text  string
	parts []domain.TextPart
}

// isUnreadable reports whether a failure to read a book is a file that holds no text to give: no
// archive, no book in it, or a book whose text is locked.
func isUnreadable(err error) bool {
	return errors.Is(err, epub.ErrNotArchive) || errors.Is(err, epub.ErrNoContainer) ||
		errors.Is(err, epub.ErrNoPackage) || errors.Is(err, epub.ErrEncrypted)
}

// readBook reads the book at a path of the vault. The book last read is kept while the file is the
// same bytes.
func (s *TextServer) readBook(path string) (bookText, error) {
	clean, err := vault.ValidatePathIn(s.vaultRoot(), path)
	if err != nil {
		return bookText{}, err
	}
	raw, err := os.ReadFile(clean)
	if err != nil {
		return bookText{}, err
	}
	hash := domain.HashContent(raw)
	s.mu.Lock()
	last := s.last
	s.mu.Unlock()
	if last.hash == hash {
		return last, nil
	}
	book, err := epub.Read(raw)
	if err != nil {
		return bookText{}, err
	}
	read := bookText{hash: hash, text: book.Text, parts: getBookParts(book.Text, book.Parts)}
	s.mu.Lock()
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText reports the text of a book as a version made from the file. A file that is no book, a
// locked book and a book with no text get no version.
func (s *TextServer) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Input != "" {
		return fmt.Errorf("%w: the EPUB reader reads files, and %s is a version", host.ErrInvalidVersion, req.Input)
	}
	read, err := s.readBook(req.Path)
	if isUnreadable(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", req.Path, err)
	}
	if strings.TrimSpace(read.text) == "" {
		return nil
	}
	_, err = s.texts.PutVersion(ctx, host.TextPut{
		Path:      req.Path,
		Key:       textKey,
		Format:    domain.TextFormatPlain,
		InputHash: read.hash,
		Hash:      domain.HashContent([]byte(read.text)),
	})
	return err
}

// GetText is one page of the text of a book. The cursor is the index of the first part.
func (s *TextServer) GetText(_ context.Context, req host.TextRequest) (host.TextPage, error) {
	if req.Key != textKey {
		return host.TextPage{}, fmt.Errorf("%w: the EPUB reader keeps no text under %q", host.ErrNotExist, req.Key)
	}
	first := 0
	if req.Cursor != "" {
		n, err := strconv.Atoi(req.Cursor)
		if err != nil || n < 0 {
			return host.TextPage{}, fmt.Errorf("cursor %q is not a part", req.Cursor)
		}
		first = n
	}
	read, err := s.readBook(req.Path)
	if isUnreadable(err) {
		return host.TextPage{}, fmt.Errorf("%w: %s has no text: %w", host.ErrNotExist, req.Path, err)
	}
	if err != nil {
		return host.TextPage{}, fmt.Errorf("read %s: %w", req.Path, err)
	}
	if strings.TrimSpace(read.text) == "" {
		return host.TextPage{}, fmt.Errorf("%w: %s has no text", host.ErrNotExist, req.Path)
	}
	if first >= len(read.parts) {
		return host.TextPage{}, fmt.Errorf("cursor %q is past the last part of %s", req.Cursor, req.Path)
	}
	last, size := first, 0
	for last < len(read.parts) && (last == first || size+len(read.parts[last].Text) <= pageBytes) {
		size += len(read.parts[last].Text)
		last++
	}
	page := host.TextPage{Parts: read.parts[first:last]}
	if last < len(read.parts) {
		page.NextCursor = strconv.Itoa(last)
	}
	return page, nil
}

// DeleteText keeps nothing to delete: the text is the book's own.
func (s *TextServer) DeleteText(context.Context, domain.VaultID, string, string) error {
	return nil
}

// getBookParts is a book's text as parts of at most partBytes, cut on whole characters, with an
// empty part opening each named section where it begins. The sections ascend by offset.
func getBookParts(text string, sections []epub.Part) []domain.TextPart {
	var parts []domain.TextPart
	from := 0
	for from < len(text) || len(sections) > 0 {
		for len(sections) > 0 && sections[0].Offset <= from {
			parts = append(parts, domain.TextPart{Section: sections[0].Title})
			sections = sections[1:]
		}
		if from >= len(text) {
			break
		}
		end := min(from+partBytes, len(text))
		to := end
		for to > from && to < len(text) && !utf8.RuneStart(text[to]) {
			to--
		}
		if to == from {
			to = end
		}
		if len(sections) > 0 && sections[0].Offset < to {
			to = sections[0].Offset
		}
		parts = append(parts, domain.TextPart{Text: text[from:to]})
		from = to
	}
	return parts
}
