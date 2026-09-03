package note

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrTooLarge is what a note over MaxBytes gets. Nothing is written. The bound
// is measured against the whole file about to be stored, which is what a read
// measures.
var ErrTooLarge = errors.New("this is more text than a note is written with")

// bounded holds a file to the most it may be: its frontmatter and its prose
// together, as they are about to go to disk. Zero holds it to nothing.
func bounded(path string, content []byte, bound int) error {
	if bound <= 0 || len(content) <= bound {
		return nil
	}
	return fmt.Errorf("write %s: %w: %d bytes, and %d is the most",
		path, ErrTooLarge, len(content), bound)
}

// ErrBodyRefused is a body that opens with the frontmatter delimiter. It is a
// whole note handed back as prose — a caller that read a file, changed it, and
// returned all of it. Writing it would put a second frontmatter block inside
// the first one's note, and the block that then reads as the note's own is the
// wrong one.
var ErrBodyRefused = errors.New(
	"a body is the prose below the frontmatter, and this one begins with a frontmatter block; " +
		"send what note_read gave you, or use the link tools to change the frontmatter")

// ErrUnreadable is a note whose frontmatter is not YAML. A write discovers it
// on the way past and leaves the file alone: repairing the block means guessing
// at what the person wrote.
var ErrUnreadable = markdown.ErrUnreadable

// ErrInline and ErrUnterminated are frontmatter one key cannot be changed in:
// a block written on one line, and a block that is never closed. Such a note
// is left alone.
var (
	ErrInline       = markdown.ErrInline
	ErrUnterminated = markdown.ErrUnterminated
)

// Write replaces the prose of a note and leaves its frontmatter alone.
//
// The two are separate on purpose. The body is what the person wrote and is
// replaced wholesale; the frontmatter is shared with them, and the keys the
// application does not own survive because nothing here rewrites them. How
// somebody's YAML is formatted stays theirs.
type Write struct {
	Readers port.VaultReaders
	Writers port.VaultWriters
	Index   func(ctx context.Context, v domain.Vault, paths []string) error
	Now     func() time.Time
	// Bound is the most the file may be, measured as it goes to disk. Zero is
	// MaxBytes. A caller whose files are read at a bound of their own sets it.
	Bound int
	// Telling is told what a write is doing while it is being made. Nothing is
	// told where nobody is drawing the note.
	Telling Telling
}

// Execute puts body in the note at path.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrChanged comes
// back: someone editing their own note outranks a caller that read it, thought
// about it, and arrived late.
//
// What comes back is the fingerprint of the file this write produced, which is
// what the caller presents at its next write.
func (u Write) Execute(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	opening := strings.TrimPrefix(body, "\ufeff")
	if strings.HasPrefix(opening, "---\n") || strings.HasPrefix(opening, "---\r\n") {
		return domain.FileRef{}, ErrBodyRefused
	}

	e := editing{
		readers: u.Readers, writers: u.Writers, index: u.Index, now: u.Now,
		fingerprint: fingerprint, bound: u.bound(),
	}
	ends := func() {}
	defer func() { ends() }()

	return e.apply(ctx, v, path, func(doc *markdown.Document) error {
		// A note rewritten whole is drawn as the stretch that changed, so what
		// a person watching sees is the change and not the note.
		was := markdown.Normalised(doc.Body())
		at, insert := markdown.Differs(was, markdown.Normalised(body))
		if at.From != at.To || insert != "" {
			ends = u.Telling.begins(ctx, domain.Editing{
				Path: path,
				From: markdown.Counted(was, at.From),
				To:   markdown.Counted(was, at.To),
				Text: insert,
			})
		}
		doc.SetBody(body)
		return nil
	})
}

// Seen is what a caller last saw of the note it is saving.
//
// Prose answers first: text that is still what the caller was given is the text
// it read, whatever the file's size and time say. A synchroniser, a checkout
// and a touch all move those over text that did not change, and comparing them
// alone is what asks a person about a file nobody edited.
//
// At answers for text that did move, and is what makes a save that follows a
// save land: the note is at the fingerprint the last write produced, so it is
// the note this caller put there.
type Seen struct {
	// Prose is what a read gave this caller, with every line break as one \n.
	Prose string
	// At is the file that read came out of.
	At domain.FileRef
}

// stale reports whether the note in front of the writer holds prose this caller
// has not read.
func (s *Seen) stale(on domain.FileRef, prose string) bool {
	if s == nil {
		return false
	}
	return prose != s.Prose && !on.Unchanged(s.At)
}

// Save puts body in the note at path, and makes the note where there is none.
//
// This is the person writing their own note in their own window. The body lands
// on the frontmatter the file holds at the moment the write goes in, and a note
// carrying no identifier keeps none. A note made here is made with no
// frontmatter at all.
//
// Seen is what the caller last saw of the note. Under the write lock the file
// is read and held against it: a note the caller has seen is written over, and
// a note holding prose it has not is left alone with port.ErrChanged. Nil is a
// caller that compares nothing, and its body lands.
//
// What comes back is the fingerprint of the file the save produced, which is
// what the caller presents at its next save.
func (u Write) Save(
	ctx context.Context, v domain.Vault, path, body string, seen *Seen,
) (domain.FileRef, error) {
	// A body opening with the delimiter is read back as a frontmatter block, and
	// then the prose it was is no longer the note's body.
	if opening := strings.TrimPrefix(body, "\ufeff"); strings.HasPrefix(opening, "---\n") ||
		strings.HasPrefix(opening, "---\r\n") {
		return domain.FileRef{}, ErrBodyRefused
	}
	e := editing{
		readers: u.Readers, writers: u.Writers, index: u.Index,
		overwrite: true,
		seen:      seen,
		bound:     u.bound(),
	}
	return e.apply(ctx, v, path, func(doc *markdown.Document) error {
		doc.SetBody(body)
		return nil
	})
}

func (u Write) bound() int {
	if u.Bound == 0 {
		return MaxBytes
	}
	return u.Bound
}
