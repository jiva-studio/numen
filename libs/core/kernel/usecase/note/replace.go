package note

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Replace puts one stretch of a note's prose in place of another, and leaves
// every byte around it as it was.
//
// What is replaced is named by the text standing there, not by at its current position,
// and the caller presents the file stat of the note it read.
type Replace struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	Drawing TellEdit
	now     machineport.Clock
}

// NewReplace creates a Replace from the vault readers and writers, the index
// updater and the clock.
// All four are required, as for NewWrite: without the index updater the index
// does not reflect the new content, and without the clock the note is stamped
// with the machine's time.
func NewReplace(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock,
) Replace {
	return Replace{readers: readers, writers: writers, index: index, now: now}
}

// ReplaceResult is what a replacement did.
type ReplaceResult struct {
	FileStat domain.FileStat
	Range    markdown.Range
	Matched  string
	IsPlain  bool
}

// MissingSpanError is a span that is not in the note, and where a copy of it stopped
// agreeing with what is there.
type MissingSpanError struct {
	Matched string
	Instead string
}

func (e MissingSpanError) Error() string {
	if e.Matched == "" {
		return "no part of this span is in the note"
	}
	return fmt.Sprintf("this span is not in the note; it holds %q where the span has %q",
		e.Instead, e.Matched+"…")
}

// AmbiguousSpanError is a span standing in more than one place, which is a
// span that does not say which of them was meant.
type AmbiguousSpanError struct {
	Places int
}

func (e AmbiguousSpanError) Error() string {
	return fmt.Sprintf("this span is in %d places; take in enough of what is around "+
		"one of them to tell it from the others", e.Places)
}

// ErrAlreadyWritten is a replacement that is already in the note and an
// original that is gone, which is the write having landed already.
var ErrAlreadyWritten = fmt.Errorf("this replacement is already in the note")

// FileStat is what the caller believes is on disk. A note that has changed
// since it was read is left alone and port.ErrStale is returned.
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
	e.Bound = edit.MaxBytes
	pos, err := e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		body := markdown.Normalize(doc.Body())

		where, plainly := markdown.FindRanges(body, current)
		if len(where) > 1 {
			return AmbiguousSpanError{Places: len(where)}
		}
		if len(where) == 0 {
			if becomes != "" && strings.Contains(body, becomes) {
				return ErrAlreadyWritten
			}
			return newMissingSpanError(body, current)
		}

		span := where[0]
		content := body[:span.Start] + becomes + body[span.End:]
		if err := doc.SetBody(content); err != nil {
			return err
		}

		// A client counts text its own way, and a span named in bytes lands
		// somewhere else in prose that is not ASCII.
		finish, tellErr := u.Drawing.beginChange(ctx, u.now, domain.Edit{
			Path: path,
			From: markdown.CountUTF16(body, span.Start),
			To:   markdown.CountUTF16(body, span.End),
			Text: becomes,
		})
		if tellErr != nil {
			return tellErr
		}
		ends = finish

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

func newMissingSpanError(body, state string) MissingSpanError {
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
		return MissingSpanError{}
	}
	end := min(first.Start+len(state), len(body))
	for end < len(body) && !utf8.RuneStart(body[end]) {
		end++
	}
	return MissingSpanError{Matched: state[:ends[count-1]], Instead: body[first.Start:end]}
}
