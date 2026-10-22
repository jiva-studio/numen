package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"

	"github.com/jiva-studio/numen/modules/libs/core/port/machineport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// ErrNoNote is what an operation gets when the vault holds no note at the path
// it was given. A vault that cannot be reached at all comes back as it arrived.
var ErrNoNote = edit.ErrNoNote

// ErrUnlevelled is a file that was written and an index that could not be
// brought up to date with it. The vault holds what the write put there, and the
// path is levelled again at the next scan.
var ErrUnlevelled = edit.ErrUnlevelled

// WrapUnlevelled is what bringing the index up to date came to, said so that a
// caller can tell it from a write that never landed. The paths are the ones the
// index was asked about.
func WrapUnlevelled(err error, paths ...string) error {
	return edit.WrapUnlevelled(err, paths...)
}

// mapMissingNote is ErrNoNote where the vault holds no note at the path, and
// the error as it arrived otherwise.
func mapMissingNote(err error) error {
	return edit.MapMissingNote(err)
}

// Baseline is what a caller last saw of the note it is saving.
type Baseline = edit.Baseline

// Levels brings the named notes up to date in the index, so that what a write
// changed is findable before the write is reported done.
type Levels = edit.Levels

// Edit is what every change to the contents of an existing note needs.
//
// It is one function because the shape is always the same and the rules in it
// are easy to forget one at a time: read, refuse what cannot be read, write the
// identifier because this is an edit, change the one thing, put it back, and
// bring the index level.
type Edit struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	now     machineport.Clock

	Fingerprint domain.Fingerprint
	// ShouldOverwrite is a caller writing what is in front of the person: the note
	// not there is made, and no identifier is stamped.
	ShouldOverwrite bool
	// Bound is the most the file may be. Zero holds it to nothing.
	Bound int
	// Baseline is what the caller last saw of the note, and is what a file that is
	// there is compared with. Nil for a caller that puts its text down whatever
	// the note now holds.
	Baseline *Baseline
}

// NewEdit is what a change to an existing note is made through: the vault it
// is read and written through, what brings it level in the index, and what time
// it is, because an edit stamps the identifier a note arrived without. What the
// caller believes is on disk, and how much of a file it will hold, are set
// beside it.
func NewEdit(
	readers vaultport.VaultReaders, writers vaultport.VaultWriters, index Levels, now machineport.Clock,
) Edit {
	return Edit{readers: readers, writers: writers, index: index, now: now}
}

// Apply makes one change to the note at path and puts it back.
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.Fingerprint, error) {
	return e.shared().Apply(ctx, v, path, change)
}

// shared is the edit as the notes module makes it, over the files of a vault.
func (e Edit) shared() edit.Edit {
	return edit.Edit{
		Open:            e.openFiles,
		Index:           e.index,
		Now:             e.now,
		Fingerprint:     e.Fingerprint,
		ShouldOverwrite: e.ShouldOverwrite,
		Bound:           e.Bound,
		Baseline:        e.Baseline,
	}
}

func (e Edit) openFiles(v domain.Vault) (edit.Files, error) {
	reader, err := e.readers.Open(v)
	if err != nil {
		return nil, err
	}
	writer, err := e.writers.Open(v)
	if err != nil {
		return nil, err
	}
	return vaultFiles{reader: reader, writer: writer, writers: e.writers, vault: v}, nil
}

// vaultFiles are the files of one vault, read and written through the ports.
type vaultFiles struct {
	reader  vaultport.VaultReader
	writer  vaultport.VaultWriter
	writers vaultport.VaultWriters
	vault   domain.Vault
}

func (f vaultFiles) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	return f.reader.GetFingerprint(ctx, path)
}

func (f vaultFiles) Read(ctx context.Context, path string) ([]byte, error) {
	return f.reader.Read(ctx, path)
}

func (f vaultFiles) Write(
	ctx context.Context, path string, content []byte, fingerprint domain.Fingerprint,
) (domain.Fingerprint, error) {
	return f.writer.Write(ctx, path, content, fingerprint)
}

func (f vaultFiles) Lock(ctx context.Context) (func(), error) {
	return f.writers.Lock(ctx, f.vault)
}
