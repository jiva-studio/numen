package indexfake

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The bytes of a book in these tests: the mark, the title, and the parts, each
// after a separator.
const (
	bookMark      = "%BOOK\n"
	bookSeparator = "\x1f"
)

// CreateBook returns the bytes of a book made of the given parts, each under
// its own heading, so the book has distinct locations.
func CreateBook(tb testing.TB, title string, parts ...string) []byte {
	tb.Helper()
	return []byte(bookMark + title + bookSeparator + strings.Join(parts, bookSeparator))
}

// BookText is a book read: its text, and the parts it names.
type BookText struct {
	Text  string
	Parts []domain.PartStart
}

// ReadBook reads the bytes CreateBook returns: each part is a heading and its
// words. It reads nothing else.
func ReadBook(raw []byte) (BookText, error) {
	if !bytes.HasPrefix(raw, []byte(bookMark)) {
		return BookText{}, source.ErrNotADocument
	}
	fields := strings.Split(strings.TrimPrefix(string(raw), bookMark), bookSeparator)
	var out BookText
	var text strings.Builder
	for i, part := range fields[1:] {
		title := fmt.Sprintf("Part %d", i+1)
		out.Parts = append(out.Parts, domain.PartStart{Title: title, Offset: text.Len()})
		text.WriteString(title + "\n" + part + "\n")
	}
	out.Text = text.String()
	return out, nil
}

// ReadBookSegments reads a book as the segments of its text version, one per
// part.
func ReadBookSegments(raw []byte) ([]domain.TextSegment, error) {
	book, err := ReadBook(raw)
	if err != nil {
		return nil, err
	}
	var parts []domain.TextSegment
	for i, part := range book.Parts {
		end := len(book.Text)
		if i+1 < len(book.Parts) {
			end = book.Parts[i+1].Offset
		}
		parts = append(parts, domain.TextSegment{Text: book.Text[part.Offset:end], Heading: part.Title})
	}
	return parts, nil
}
