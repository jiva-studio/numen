package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Write replaces the prose of a note and leaves its frontmatter unchanged.
//
// The body belongs to the user and is replaced entirely. The frontmatter is
// shared with the user, and keys the application does not own are preserved
// because nothing here rewrites them. The user's YAML formatting is retained.
type Write struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	now     machineport.Clock
	Bound   int
	Drawing TellEdit
	gate    WriteGate
}

// NewWrite returns the writer that saves a note to disk. It takes the vault
// readers and writers, the index synchronizer, the clock, and the gate a write passes.
func NewWrite(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock, gate WriteGate,
) Write {
	return Write{readers: readers, writers: writers, index: index, now: now, gate: gate}
}

// Execute writes body to the note at path. It fails with window.ErrClosing once the window is closing.
// FileStat, when set, is the file state the caller expects on disk. A note
// changed since it was read is left unchanged and port.ErrStale is returned:
// the user's edits take precedence over a caller working from an older read.
// The returned file stat describes the file this write produced.
func (u Write) Execute(
	ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat,
) (domain.FileStat, error) {
	return writeUnder(u.gate, func() (domain.FileStat, error) { return u.replace(ctx, v, path, body, stat) })
}

func (u Write) replace(
	ctx context.Context, v domain.Vault, path, body string, stat domain.FileStat,
) (result domain.FileStat, err error) {
	if err := domain.CheckPath(path); err != nil {
		return domain.FileStat{}, err
	}
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, edit.ErrBodyUnwritable
	}

	e := NewEdit(u.readers, u.writers, u.index, u.now)
	e.FileStat = stat
	e.Bound = u.maxBound()
	ends := func() error { return nil }
	defer func() { err = errors.Join(err, ends()) }()

	return e.Apply(ctx, v, path, func(doc *markdown.Document) error {
		// A note rewritten whole is rendered as the changed range, so an
		// observer sees the change and not the whole note.
		was := markdown.Normalize(doc.Body())
		pos, insert := markdown.Diff(was, markdown.Normalize(body))
		// watching should see change.
		if err := doc.SetBody(body); err != nil {
			return err
		}
		if pos.Start != pos.End || insert != "" {
			finish, tellErr := u.Drawing.beginChange(ctx, u.now, domain.Edit{
				Path: path,
				From: markdown.CountUTF16(was, pos.Start),
				To:   markdown.CountUTF16(was, pos.End),
				Text: insert,
			})
			if tellErr != nil {
				return tellErr
			}
			ends = finish
		}
		return nil
	})
}

// Save writes body to the note at path, creating the note if it does not exist.
// This is the user saving their own note. The body is written onto the
// frontmatter the file holds when the write begins; a note without an
// identifier keeps none, and a created note has no frontmatter.
// Baseline is the state the caller last read: a differing note is left unchanged and port.ErrStale is returned; nil skips the check.
// It fails with window.ErrClosing once the window is closing.
func (u Write) Save(
	ctx context.Context, v domain.Vault, path, body string, baseline *edit.Baseline,
) (domain.FileStat, error) {
	return writeUnder(u.gate, func() (domain.FileStat, error) { return u.overwrite(ctx, v, path, body, baseline) })
}

func (u Write) overwrite(
	ctx context.Context, v domain.Vault, path, body string, baseline *edit.Baseline,
) (domain.FileStat, error) {
	if err := domain.CheckPath(path); err != nil {
		return domain.FileStat{}, err
	}
	if markdown.IsFrontmatterStart(body) {
		return domain.FileStat{}, edit.ErrBodyUnwritable
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
		return edit.MaxBytes
	}
	return u.Bound
}
