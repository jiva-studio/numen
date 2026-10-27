package service

import (
	"context"
	"errors"
	"fmt"
	"os"
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

// partBytes is the most text one segment holds.
const partBytes = 64 << 10

// textKind is the kind of the text versions the plugin makes.
const textKind domain.TextKind = "book-text"

// TextServer writes the text of the vault's EPUB books into the plugin's folder as text versions,
// each named section opened where it begins. The text and its offsets are the ones the reader sets
// the book with.
type TextServer struct {
	vaultRoot func() string
	texts     host.Texts
	store     host.Store

	mu   sync.Mutex
	last bookText
}

var _ host.TextProvider = (*TextServer)(nil)

// NewTextServer is the texts of the books of the vault at vaultRoot, written into store and
// reported to texts.
func NewTextServer(vaultRoot func() string, texts host.Texts, store host.Store) *TextServer {
	return &TextServer{vaultRoot: vaultRoot, texts: texts, store: store}
}

// bookText is one file's text as segments, and the hash of the bytes it was read from.
type bookText struct {
	hash     string
	text     string
	segments []domain.TextSegment
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
	read := bookText{hash: hash, text: book.Text, segments: getBookParts(book.Text, book.Parts)}
	s.mu.Lock()
	s.last = read
	s.mu.Unlock()
	return read, nil
}

// CreateText reports the text of a book as a version made from the file. A file that is no book, a
// locked book and a book with no text get no version.
func (s *TextServer) CreateText(ctx context.Context, req host.TextCreation) error {
	if req.Source != "" {
		return fmt.Errorf("%w: the EPUB reader reads files, and %s is a version", host.ErrInvalidVersion, req.Source)
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
	store := s.store
	if req.Vault != "" {
		store = store.With(req.Vault)
	}
	file := read.hash + ".jsonl"
	if err := host.WriteText(ctx, store, file, read.segments); err != nil {
		return err
	}
	_, err = s.texts.PutVersion(ctx, host.TextPut{
		Path: req.Path, Key: textKey, Kind: textKind, SourceHash: read.hash, File: file,
	})
	return err
}

// getBookParts is a book's text as segments of at most partBytes, cut on whole characters, with an
// empty segment opening each named section where it begins. The sections ascend by offset.
func getBookParts(text string, sections []epub.Part) []domain.TextSegment {
	var parts []domain.TextSegment
	from := 0
	for from < len(text) || len(sections) > 0 {
		for len(sections) > 0 && sections[0].Offset <= from {
			parts = append(parts, domain.TextSegment{Heading: sections[0].Title})
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
		parts = append(parts, domain.TextSegment{Text: text[from:to]})
		from = to
	}
	return parts
}
