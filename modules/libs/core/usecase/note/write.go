package note

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// ErrTooLarge is what a note over MaxBytes gets. Nothing is written. The bound
// is measured against the whole file about to be stored, which is what a read
// measures.
var ErrTooLarge = errors.New("this is more text than a note is written with")

// Bounded holds a file to the most it may be: its frontmatter and its prose
// together, as they are about to go to disk. Zero holds it to nothing.
func Bounded(path string, size, bound int) error {
	if bound <= 0 || size <= bound {
		return nil
	}
	return fmt.Errorf("write %s: %w: %d bytes, and %d is the most",
		path, ErrTooLarge, size, bound)
}

// ErrBodyRefused is a body that opens with the frontmatter delimiter.
var ErrBodyRefused = markdown.ErrBodyRefused

// ErrUnreadable is a note whose frontmatter is not YAML. A write discovers it
// on the way past and leaves the file alone: repairing the block means guessing
// at what the person wrote.
var ErrUnreadable = markdown.ErrUnreadable

// ErrInline, ErrUnterminated and ErrAnchored are frontmatter one key cannot be
// changed in: a block written on one line, a block that is never closed, and a
// block carrying a YAML anchor. Such a note is left alone.
var (
	ErrInline       = markdown.ErrInline
	ErrUnterminated = markdown.ErrUnterminated
	ErrAnchored     = markdown.ErrAnchored
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
	Index   Levels
	Now     port.Clock
	// Bound is the most the file may be, measured as it goes to disk. Zero is
	// MaxBytes. A caller whose files are read at a bound of their own sets it.
	Bound int
	// Drawing is told what a write is doing while it is being made. Nothing is
	// told where nobody is drawing the note.
	Drawing TellEdit
}

// Levels brings the named notes up to date in the index, so that what a write
// changed is findable before the write is reported done.
type Levels func(ctx context.Context, v domain.Vault, paths []string) error

// NewWrite is the writer a note somebody is saving goes to disk through: the
// vault it is read and written through, what brings it level in the index, and
// what time it is.
//
// All four are named here because a write without any one of them is a save
// that half happens. A caller that forgets the levelling or the clock does not
// compile, where a struct built field by field would save the note and quietly
// leave the vault unable to find what it now holds, or stamp it off the clock
// of whichever machine happened to be running.
func NewWrite(
	readers port.VaultReaders, writers port.VaultWriters, index Levels, now port.Clock,
) Write {
	return Write{Readers: readers, Writers: writers, Index: index, Now: now}
}

// Execute puts body in the note at path.
//
// Fingerprint, when it is given, is what the caller believes is on disk. A note
// that has changed since it was read is left alone and port.ErrStale comes
// back: someone editing their own note outranks a caller that read it, thought
// about it, and arrived late.
//
// What comes back is the fingerprint of the file this write produced, which is
// what the caller presents at its next write.
func (u Write) Execute(
	ctx context.Context, v domain.Vault, path, body string, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	if markdown.OpensFrontmatter(body) {
		return domain.Fingerprint{}, ErrBodyRefused
	}

	e := Edit{
		Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now,
		Fingerprint: fingerprint, Bound: u.bound(),
	}
	ends := func() {}
	defer func() { ends() }()

	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		// A note rewritten whole is drawn as the stretch that changed, so what
		// a person watching sees is the change and not the note.
		was := markdown.Normalised(doc.Body())
		at, insert := markdown.Differs(was, markdown.Normalised(body))
		// Told after the write is settled: a refusal is not a stretch anybody
		// watching should see change.
		if err := doc.SetBody(body); err != nil {
			return err
		}
		if at.From != at.To || insert != "" {
			ends = u.Drawing.begins(ctx, u.Now, domain.Edit{
				Path: path,
				From: markdown.Counted(was, at.From),
				To:   markdown.Counted(was, at.To),
				Text: insert,
			})
		}
		return nil
	})
}

// LastRead is what a caller last saw of the note it is saving.
//
// Prose answers first: text that is still what the caller was given is the text
// it read, whatever the file's size and time say. A synchroniser, a checkout
// and a touch all move those over text that did not change, and comparing them
// alone is what asks a person about a file nobody edited.
//
// Fingerprint answers for text that did move, and is what makes a save that
// follows a save land: the note is at the fingerprint the last write produced,
// so it is the note this caller put there.
type LastRead struct {
	// Prose is what a read gave this caller, with every line break as one \n.
	Prose string
	// Fingerprint is the file that read came out of.
	Fingerprint domain.Fingerprint
}

// stale reports whether the note in front of the writer holds prose this caller
// has not read.
func (s *LastRead) stale(on domain.Fingerprint, prose string) bool {
	if s == nil {
		return false
	}
	return prose != s.Prose && !on.Unchanged(s.Fingerprint)
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
// a note holding prose it has not is left alone with port.ErrStale. Nil is a
// caller that compares nothing, and its body lands.
//
// What comes back is the fingerprint of the file the save produced, which is
// what the caller presents at its next save.
func (u Write) Save(
	ctx context.Context, v domain.Vault, path, body string, seen *LastRead,
) (domain.Fingerprint, error) {
	if markdown.OpensFrontmatter(body) {
		return domain.Fingerprint{}, ErrBodyRefused
	}
	e := Edit{
		Readers: u.Readers, Writers: u.Writers, Index: u.Index, Now: u.Now,
		Overwrite: true,
		Seen:      seen,
		Bound:     u.bound(),
	}
	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		return doc.SetBody(body)
	})
}

func (u Write) bound() int {
	if u.Bound == 0 {
		return MaxBytes
	}
	return u.Bound
}
