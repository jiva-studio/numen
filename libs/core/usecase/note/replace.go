package note

import (
	"context"
	"errors"
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
// and the caller presents the file stat of the note it read.
type Replace struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	// Drawing is notified of each change while it is applied. When nil, no
	// notification is sent.
	Drawing TellEdit
	now     machineport.Clock
}

// NewReplace creates a Replace from the vault readers and writers, the index
// leveller and the clock.
//
// All four are required, as for NewWrite: without the leveller, a replacement
// changes the file but the index does not reflect the new content, and without
// the clock the note is stamped with the machine's time.
func NewReplace(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels, now machineport.Clock,
) Replace {
	return Replace{readers: readers, writers: writers, index: index, now: now}
}

// ReplaceResult is what a replacement did.
type ReplaceResult struct {
	// FileStat is of the file this write produced.
	FileStat domain.FileStat
	// Range is the location of the span, as byte offsets into the prose that a
	// read returns.
	Range markdown.Range
	// Matched is the span as stored in the note, which can differ from the text
	// the caller requested.
	Matched string
	// IsPlain reports that the span matched only after ignoring differences in
	// punctuation or spacing.
	IsPlain bool
}

// MissingSpan is a span that is not in the note, and where a copy of it stopped
// agreeing with what is there.
type MissingSpan struct {
	// Matched is the longest prefix of the requested span that is present in the
	// note.
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

// FileStat is what the caller believes is on disk. A note that has changed
// since it was read is left alone and port.ErrStale comes back.
func (u Replace) Execute(
	ctx context.Context, v domain.Vault, path, current, becomes string, stat domain.FileStat,
) (result ReplaceResult, err error) {
	if current == "" {
		return ReplaceResult{}, fmt.Errorf("name the text to replace")
	}

	isDone := ReplaceResult{}
	ends := func() error { return nil }
	defer func() { err = errors.Join(err, ends()) }()

	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat = stat
	e.Bound = MaxBytes
	pos, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		body := markdown.Normalise(doc.Body())

		where, plainly := markdown.FindRanges(body, current)
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
		content := body[:span.Start] + becomes + body[span.End:]
		if err := doc.SetBody(content); err != nil {
			return err
		}

		// A client counts text its own way, and a span named in bytes lands
		// somewhere else in prose that is not ASCII.
		var tellErr error
		ends, err = u.Drawing.beginChange(ctx, u.now, domain.Edit{
			Path: path,
			From: markdown.CountUTF16(body, span.Start),
			To:   markdown.CountUTF16(body, span.End),
			Text: becomes,
		})
		if tellErr != nil {
			return tellErr
		}

		isDone.Range = markdown.Range{Start: span.Start, End: span.Start + len(becomes)}
		isDone.Matched = body[span.Start:span.End]
		isDone.IsPlain = plainly
		return nil
	})
	if err != nil {
		return ReplaceResult{}, err
	}
	isDone.FileStat = pos
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
	var first markdown.Range
	count := sort.Search(len(ends), func(i int) bool {
		offset, _ := markdown.FindRanges(body, state[:ends[i]])
		if len(offset) == 0 {
			return true
		}
		first = offset[0]
		return false
	})
	if count == 0 {
		return MissingSpan{}
	}
	end := min(first.Start+len(state), len(body))
	for end < len(body) && !utf8.RuneStart(body[end]) {
		end++
	}
	return MissingSpan{Matched: state[:ends[count-1]], Instead: body[first.Start:end]}
}
