package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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

// ErrUnreadable is a note whose frontmatter is not YAML. A write detects it and
// leaves the file unchanged: repairing the block would mean guessing the user's
// content.
var ErrUnreadable = edit.ErrUnreadable

// ErrInline, ErrUnterminated and ErrAnchored are frontmatter one key cannot be
// changed in: a block written on one line, a block that is never closed, and a
// block carrying a YAML anchor. Such a note is left alone.
var (
	ErrInline       = edit.ErrInline
	ErrUnterminated = edit.ErrUnterminated
	ErrAnchored     = edit.ErrAnchored
)

// Write replaces the prose of a note and leaves its frontmatter unchanged.
//
// The body belongs to the user and is replaced entirely. The frontmatter is
// shared with the user, and keys the application does not own are preserved
// because nothing here rewrites them. The user's YAML formatting is retained.
type Write struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	now     machineport.Clock
	// Bound is the most the file may be, measured as it goes to disk. Zero is
	// MaxBytes. A caller whose files are read at a bound of their own sets it.
	Bound int
	// Drawing receives the edit being applied while a write is in progress. It
	// is unset when no viewer renders the note.
	Drawing TellEdit
}

// NewWrite returns the writer that saves a note to disk. It takes the vault
// readers and writers, the index synchronizer, and the clock.
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

// Execute writes body to the note at path.
//
// FileStat, when set, is the file state the caller expects on disk. A note
// changed since it was read is left unchanged and port.ErrStale is returned:
// the user's edits to their own note take precedence over a caller working from
// an older read.
//
// The returned file stat describes the file this write produced, which the
// caller presents at its next write.
func (u Write) Execute(
	ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat,
) (result domain.FileStat, err error) {
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, ErrBodyUnwritable
	}

	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat = stat
	e.Bound = u.maxBound()
	ends := func() error { return nil }
	defer func() { err = errors.Join(err, ends()) }()

	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		// A note rewritten whole is rendered as the changed range, so an
		// observer sees the change and not the whole note.
		was := markdown.Normalise(doc.Body())
		pos, insert := markdown.Diff(was, markdown.Normalise(body))
		// watching should see change.
		if err := doc.SetBody(body); err != nil {
			return err
		}
		if pos.Start != pos.End || insert != "" {
			var tellErr error
			ends, tellErr = u.Drawing.beginChange(ctx, u.now, domain.Edit{
				Path: path,
				From: markdown.CountUTF16(was, pos.Start),
				To:   markdown.CountUTF16(was, pos.End),
				Text: insert,
			})
			if tellErr != nil {
				return tellErr
			}
		}
		return nil
	})
}

// Save writes body to the note at path, creating the note if it does not exist.
//
// This is the user saving their own note in the window. The body is written
// onto the frontmatter the file holds when the write begins, and a note without
// an identifier keeps none. A note created here has no frontmatter.
//
// Baseline is the state the caller last read. Under the write lock the file is
// compared with it: a note matching the baseline is overwritten, and a note
// whose prose differs is left unchanged and port.ErrStale is returned. A nil
// Baseline skips the comparison and the body is written.
//
// The returned file stat describes the file the save produced, which the caller
// presents at its next save.
func (u Write) Save(
	ctx context.Context, v domain.Vault, path, body string, baseline *Baseline,
) (domain.FileStat, error) {
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, ErrBodyUnwritable
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
