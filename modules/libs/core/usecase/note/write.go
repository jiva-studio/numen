package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// ErrTooLarge is what a note over MaxBytes gets. Nothing is written. The bound
// is measured against the whole file about to be stored, which is what a read
// measures.
var ErrTooLarge = edit.ErrTooLarge

// CheckSize holds a file to the most it may be: its frontmatter and its prose
// together, as they are about to go to disk. Zero holds it to nothing.
func CheckSize(path string, size, limit int) error {
	return edit.CheckSize(path, size, limit)
}

// ErrBodyUnwritable is a body that opens with the frontmatter delimiter.
var ErrBodyUnwritable = edit.ErrBodyUnwritable

// ErrUnreadable is a note whose frontmatter is not YAML. A write discovers it
// on the way past and leaves the file alone: repairing the block means guessing
// at what the person wrote.
var ErrUnreadable = edit.ErrUnreadable

// ErrInline, ErrUnterminated and ErrAnchored are frontmatter one key cannot be
// changed in: a block written on one line, a block that is never closed, and a
// block carrying a YAML anchor. Such a note is left alone.
var (
	ErrInline       = edit.ErrInline
	ErrUnterminated = edit.ErrUnterminated
	ErrAnchored     = edit.ErrAnchored
)

// Write replaces the prose of a note and leaves its frontmatter alone.
//
// The two are separate on purpose. The body is what the person wrote and is
// replaced wholesale; the frontmatter is shared with them, and the keys the
// application does not own survive because nothing here rewrites them. How
// somebody's YAML is formatted stays theirs.
type Write struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	now     machineport.Clock
	// Bound is the most the file may be, measured as it goes to disk. Zero is
	// MaxBytes. A caller whose files are read at a bound of their own sets it.
	Bound int
	// Drawing is told what a write is doing while it is being made. Nothing is
	// told where nobody is drawing the note.
	Drawing TellEdit
}

// NewWrite is the writer a note somebody is saving goes to disk through: the
// vault it is read and written through, what brings it level in the index, and
// what time it is.
func NewWrite(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels, now machineport.Clock,
) Write {
	return Write{readers: readers, writers: writers, index: index, now: now}
}

// WithReaders replaces the vault readers.
func (u Write) WithReaders(readers vaultport.VaultReaders) Write {
	u.readers = readers
	return u
}

// WithWriters replaces the vault writers.
func (u Write) WithWriters(writers vaultport.VaultWriters) Write {
	u.writers = writers
	return u
}

// WithIndex replaces the index function.
func (u Write) WithIndex(index Levels) Write {
	u.index = index
	return u
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
	if markdown.IsFrontmatterStart(body) {
		return domain.Fingerprint{}, ErrBodyUnwritable
	}

	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.Fingerprint = fingerprint
	e.Bound = u.maxBound()
	ends := func() {}
	defer func() { ends() }()

	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		// A note rewritten whole is drawn as the stretch that changed, so what
		// a person watching sees is the change and not the note.
		was := markdown.Normalise(doc.Body())
		pos, insert := markdown.Diff(was, markdown.Normalise(body))
		// watching should see change.
		if err := doc.SetBody(body); err != nil {
			return err
		}
		if pos.From != pos.To || insert != "" {
			ends = u.Drawing.beginChange(ctx, u.now, domain.Edit{
				Path: path,
				From: markdown.CountUTF16(was, pos.From),
				To:   markdown.CountUTF16(was, pos.To),
				Text: insert,
			})
		}
		return nil
	})
}

// Save puts body in the note at path, and makes the note where there is none.
//
// This is the person writing their own note in their own window. The body lands
// on the frontmatter the file holds at the moment the write goes in, and a note
// carrying no identifier keeps none. A note made here is made with no
// frontmatter at all.
//
// Baseline is what the caller last saw of the note. Under the write lock the file
// is read and held against it: a note the caller has baseline is written over, and
// a note holding prose it has not is left alone with port.ErrStale. Nil is a
// caller that compares nothing, and its body lands.
//
// What comes back is the fingerprint of the file the save produced, which is
// what the caller presents at its next save.
func (u Write) Save(
	ctx context.Context, v domain.Vault, path, body string, baseline *Baseline,
) (domain.Fingerprint, error) {
	if markdown.IsFrontmatterStart(body) {
		return domain.Fingerprint{}, ErrBodyUnwritable
	}
	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.ShouldOverwrite = true
	e.Baseline = baseline
	e.Bound = u.maxBound()
	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		return doc.SetBody(body)
	})
}

func (u Write) maxBound() int {
	if u.Bound == 0 {
		return MaxBytes
	}
	return u.Bound
}
