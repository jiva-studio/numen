package edit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

// ReadOutcome is how a read ended. Each one leaves the caller with something different to do: a
// missing note is created by writing it, a note that is too large is opened in something else, and
// a file that is not a note or not text is left alone.
type ReadOutcome string

const (
	// Ok is the returned body.
	Ok ReadOutcome = "ok"
	// Missing is a path with no file behind it.
	Missing ReadOutcome = "missing"
	// NotANote is a file the vault does not hold as one: an attachment, an export, whatever the
	// vault's own rules leave alone.
	NotANote ReadOutcome = "not a note"
	// NotText is a file that is not valid UTF-8.
	NotText ReadOutcome = "not text"
	// TooLarge is a file over the bound.
	TooLarge ReadOutcome = "too large"
	// Unreadable is a note whose frontmatter cannot be read. Such a note can be neither read nor
	// written from here.
	Unreadable ReadOutcome = "unreadable"
)

// Reader is the file access a read requires.
type Reader interface {
	// StatFile reports what the file is without opening it. It returns fs.ErrNotExist for a file
	// that is missing and ErrNotANote for one the vault does not hold as a note.
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the bytes of the file.
	Read(ctx context.Context, path string) ([]byte, error)
}

// Contents is one note as returned by a read.
type Contents struct {
	Path     string
	Outcome  ReadOutcome
	Body     string
	FileStat domain.FileStat
}

// Read returns the prose of the note at path, rejecting files larger than MaxBytes.
//
// An error means the vault is unreachable. A fault in the note itself is an outcome, and
// the caller receives which one.
func Read(ctx context.Context, files Reader, path string) (Contents, error) {
	out := Contents{Path: path}

	target, stop, err := statNote(ctx, files, path, &out)
	if err != nil {
		return Contents{}, err
	}
	if stop {
		return out, nil
	}

	raw, stop, err := readNoteBytes(ctx, files, path, target, &out)
	if err != nil {
		return Contents{}, err
	}
	if stop {
		return out, nil
	}

	// gomutants:disable-next-line BRANCH_IF reason="readBody handles every error returned by markdown.Open via Outcomes"
	if err := readBody(raw, &out); err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL reason="readBody never returns a non-nil error"
		return Contents{}, err
	}
	return out, nil
}

func statNote(ctx context.Context, files Reader, path string, out *Contents) (target, stop bool, err error) {
	ref, err := files.StatFile(ctx, path)
	switch {
	case err == nil:
		out.FileStat = ref
		// A vault holds several kinds of source and this reads one of them.
		if ref.Kind != domain.KindNote {
			out.Outcome = NotANote
			// gomutants:disable-next-line RETURN_FALSE reason="Read stops immediately when stop is true and never inspects target"
			return true, true, nil
		}
		if ref.Size > MaxBytes {
			out.Outcome = TooLarge
			// gomutants:disable-next-line RETURN_FALSE reason="Read stops immediately when stop is true and never inspects target"
			return true, true, nil
		}
		return true, false, nil
	case errors.Is(err, ErrNotANote):
		out.Outcome = NotANote
		return false, true, nil
	case !errors.Is(err, fs.ErrNotExist):
		return false, false, fmt.Errorf("look at %s: %w", path, err)
	}
	return false, false, nil
}

func readNoteBytes(ctx context.Context, files Reader, path string, target bool, out *Contents) (raw []byte, stop bool, err error) {
	raw, err = files.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = Missing
		return nil, true, nil
	case !target && err == nil:
		out.Outcome = NotANote
		return nil, true, nil
	case err != nil:
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	return raw, false, nil
}

func readBody(raw []byte, out *Contents) error {
	// The body is returned in a string field. A browser converts an ill-formed sequence to U+FFFD, and
	// the next save would write those characters in place of the original bytes.
	if !utf8.Valid(raw) {
		out.Outcome = NotText
		return nil
	}

	doc, err := markdown.Open(raw)
	if errors.Is(err, markdown.ErrUnreadable) {
		out.Outcome = Unreadable
		return nil
	}
	// gomutants:disable-next-line BRANCH_IF reason="markdown.Open wraps every error it returns in ErrUnreadable"
	if err != nil {
		// gomutants:disable-next-line RETURN_ERROR_NIL,ERRORF_WRAP reason="markdown.Open wraps every error it returns in ErrUnreadable"
		return fmt.Errorf("read %s: %w", out.Path, err)
	}
	out.Body = markdown.Normalize(doc.Body())
	out.Outcome = Ok
	return nil
}
