package note

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Replace puts one stretch of a note's prose in place of another, and leaves
// every byte around it as it was.
//
// What is replaced is named by the text standing there, not by at its current position,
// and the caller presents the fingerprint of the note it read.
type Replace struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	// Drawing is told what this change is doing while it is being made. Nothing
	// is told where nobody is drawing the note.
	Drawing TellEdit
	now     machineport.Clock
}

// NewReplace is what one stretch of a note is put right through: the vault it
// is read and written through, what brings it level in the index, and what time
// it is.
//
// All four are named here for the reason NewWrite names them: a replacement
// short of the levelling changes the file and leaves the vault unable to find
// what it now says, and one short of the clock stamps the note off the
// machine's.
func NewReplace(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels, now machineport.Clock,
) Replace {
	return Replace{readers: readers, writers: writers, index: index, now: now}
}

// ReplaceResult is what a replacement did.
type ReplaceResult struct {
	// Fingerprint is of the file this write produced.
	Fingerprint domain.Fingerprint
	// Span is where the span stood, as byte offsets into the prose a read hands
	// out.
	Span markdown.Span
	// Matched is the span as the note held it, which is not always the text the
	// caller asked for.
	Matched string
	// IsPlain says the span was found only once punctuation or spacing were
	// allowed to differ.
	IsPlain bool
}

// MissingSpan is a span that is not in the note, and where a copy of it stopped
// agreeing with what is there.
type MissingSpan struct {
	// Matched is the longest opening of what was asked for that does stand in
	Matched string
	Instead string
}

func (e MissingSpan) Error() string {
	if e.Matched == "" {
		return "no part of this span is in the note"
	}
	return fmt.Sprintf("this span is not in the note; it holds %q where the span has %q",
		e.Instead, e.Matched+"…")
}

// AmbiguousSpan is a span standing in more than one place, which is a
// span that does not say which of them was meant.
type AmbiguousSpan struct {
	Places int
}

func (e AmbiguousSpan) Error() string {
	return fmt.Sprintf("this span stands in %d places; take in enough of what is around "+
		"one of them to tell it from the others", e.Places)
}

// ErrAlreadyWritten is a replacement that is already in the note and an
// original that is gone, which is the write having landed already.
var ErrAlreadyWritten = fmt.Errorf("this replacement is already in the note")

// Fingerprint is what the caller believes is on disk. A note that has changed
// since it was read is left alone and port.ErrStale comes back.
func (u Replace) Execute(
	ctx context.Context, v domain.Vault, path, current, becomes string, fingerprint domain.Fingerprint,
) (ReplaceResult, error) {
	if current == "" {
		return ReplaceResult{}, fmt.Errorf("name the text to replace")
	}

	isDone := ReplaceResult{}
	ends := func() {}
	defer func() { ends() }()

	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.Fingerprint = fingerprint
	e.Bound = MaxBytes
	pos, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		body := markdown.Normalise(doc.Body())

		where, plainly := markdown.FindSpans(body, current)
		if len(where) > 1 {
			return AmbiguousSpan{Places: len(where)}
		}
		if len(where) == 0 {
			if becomes != "" && strings.Contains(body, becomes) {
				return ErrAlreadyWritten
			}
			return newMissingSpan(body, current)
		}

		span := where[0]
		content := body[:span.From] + becomes + body[span.To:]
		if err := doc.SetBody(content); err != nil {
			return err
		}

		// A client counts text its own way, and a span named in bytes lands
		// somewhere else in prose that is not ASCII.
		ends = u.Drawing.beginChange(ctx, u.now, domain.Edit{
			Path: path,
			From: markdown.CountUTF16(body, span.From),
			To:   markdown.CountUTF16(body, span.To),
			Text: becomes,
		})

		isDone.Span = markdown.Span{From: span.From, To: span.From + len(becomes)}
		isDone.Matched = body[span.From:span.To]
		isDone.IsPlain = plainly
		return nil
	})
	if err != nil {
		return ReplaceResult{}, err
	}
	isDone.Fingerprint = pos
	return isDone, nil
}

// newMissingSpan is what to say about a span that is not in the note: how much of
//
// The opening is found by halving over the ends of its runes, which the text being
// present for every shorter opening allows.
func newMissingSpan(body, state string) MissingSpan {
	var ends []int
	for pos := 0; pos < len(state); {
		_, width := utf8.DecodeRuneInString(state[pos:])
		pos += width
		ends = append(ends, pos)
	}
	var first markdown.Span
	count := sort.Search(len(ends), func(i int) bool {
		offset, _ := markdown.FindSpans(body, state[:ends[i]])
		if len(offset) == 0 {
			return true
		}
		first = offset[0]
		return false
	})
	if count == 0 {
		return MissingSpan{}
	}
	end := min(first.From+len(state), len(body))
	for end < len(body) && !utf8.RuneStart(body[end]) {
		end++
	}
	return MissingSpan{Matched: state[:ends[count-1]], Instead: body[first.From:end]}
}
