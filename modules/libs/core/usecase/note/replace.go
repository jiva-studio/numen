package note

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Replace puts one stretch of a note's prose in place of another, and leaves
// every byte around it as it was.
//
// What is replaced is named by the text standing there, not by where it stands,
// and the caller presents the fingerprint of the note it read.
type Replace struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	// Telling is told what this change is doing while it is being made. Nothing
	// is told where nobody is drawing the note.
	Telling TellEditing
	Now     func() time.Time
}

// ReplaceResult is what a replacement did.
type ReplaceResult struct {
	// At is the fingerprint of the file this write produced.
	At domain.Fingerprint
	// Span is where the span stood, as byte offsets into the prose a read hands
	// out.
	Span markdown.Span
	// Stood is the span as the note held it, which is not always the text the
	// caller asked for.
	Stood string
	// Plainly says the span was found only once punctuation or spacing were
	// allowed to differ.
	Plainly bool
}

// MissingStretch is a stretch that is not in the note, and where a copy of it stopped
// agreeing with what is there.
type MissingStretch struct {
	// Matched is the longest opening of what was asked for that does stand in
	// the note, and Instead is what stands in the note from there.
	Matched string
	Instead string
}

func (e MissingStretch) Error() string {
	if e.Matched == "" {
		return "no part of this stretch is in the note"
	}
	return fmt.Sprintf("this stretch is not in the note; it holds %q where the stretch has %q",
		e.Instead, e.Matched+"…")
}

// Twice is a stretch standing in more than one place, which is a stretch that
// does not say which of them was meant.
type Twice struct {
	Places int
}

func (e Twice) Error() string {
	return fmt.Sprintf("this stretch stands in %d places; take in enough of what is around "+
		"one of them to tell it from the others", e.Places)
}

// ErrAlreadyWritten is a replacement that is already in the note and an
// original that is gone, which is the write having landed already.
var ErrAlreadyWritten = fmt.Errorf("this replacement is already in the note")

// Execute puts `becomes` where `stood` stands in the note at path.
//
// Fingerprint is what the caller believes is on disk. A note that has changed
// since it was read is left alone and port.ErrChanged comes back.
func (u Replace) Execute(
	ctx context.Context, v domain.Vault, path, stood, becomes string, fingerprint domain.Fingerprint,
) (ReplaceResult, error) {
	if stood == "" {
		return ReplaceResult{}, fmt.Errorf("name the text to replace")
	}

	done := ReplaceResult{}
	ends := func() {}
	defer func() { ends() }()

	e := editing{
		readers: u.Readers, writers: u.Writers, index: u.Index, now: u.Now,
		fingerprint: fingerprint, bound: MaxBytes,
	}
	at, err := e.apply(ctx, v, path, func(doc *markdown.Document) error {
		body := markdown.Normalised(doc.Body())

		where, plainly := markdown.Where(body, stood)
		switch {
		case len(where) == 1:
		case len(where) > 1:
			return Twice{Places: len(where)}
		case becomes != "" && strings.Contains(body, becomes):
			return ErrAlreadyWritten
		default:
			return nowhere(body, stood)
		}

		span := where[0]
		written := body[:span.From] + becomes + body[span.To:]

		// A client counts text its own way, and a span named in bytes lands
		// somewhere else in prose that is not ASCII.
		ends = u.Telling.begins(ctx, domain.Edit{
			Path: path,
			From: markdown.Counted(body, span.From),
			To:   markdown.Counted(body, span.To),
			Text: becomes,
		})

		done.Span = markdown.Span{From: span.From, To: span.From + len(becomes)}
		done.Stood = body[span.From:span.To]
		done.Plainly = plainly
		doc.SetBody(written)
		return nil
	})
	if err != nil {
		return ReplaceResult{}, err
	}
	done.At = at
	return done, nil
}

// nowhere is what to say about a stretch that is not in the note: how much of
// its opening does stand there, and what stands in its place.
//
// The opening is found by halving, which the text being present for every
// shorter opening allows.
func nowhere(body, stood string) MissingStretch {
	low, high := 0, len(stood)
	for low < high {
		middle := low + (high-low+1)/2
		for middle < high && !utf8.RuneStart(stood[middle]) {
			middle++
		}
		if middle > high {
			break
		}
		if at, _ := markdown.Where(body, stood[:middle]); len(at) > 0 {
			low = middle
			continue
		}
		high = middle - 1
		for high > low && !utf8.RuneStart(stood[high]) {
			high--
		}
	}
	if low == 0 {
		return MissingStretch{}
	}
	matched := stood[:low]
	at, _ := markdown.Where(body, matched)
	if len(at) == 0 {
		return MissingStretch{}
	}
	end := min(at[0].From+len(stood), len(body))
	for end < len(body) && !utf8.RuneStart(body[end]) {
		end++
	}
	return MissingStretch{Matched: matched, Instead: body[at[0].From:end]}
}
