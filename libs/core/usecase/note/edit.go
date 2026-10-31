package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/machineport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// ErrNoNote is what an operation gets when the vault holds no note at the path
// it was given. A vault that cannot be reached at all comes back as it arrived.
var ErrNoNote = edit.ErrNoNote

// ErrUnlevelled is returned when a file was written but the index could not be
// updated. The vault holds the written content, and the next scan updates the
// index for the path.
var ErrUnlevelled = edit.ErrUnlevelled

// WrapUnlevelled wraps an index update failure in ErrUnlevelled, which
// distinguishes it from a write that failed. The paths are the ones the index
// update covered.
func WrapUnlevelled(err error, paths ...string) error {
	return edit.WrapUnlevelled(err, paths...)
}

// mapMissingNote returns ErrNoNote when the vault holds no note at the path,
// and err unchanged for any other error.
func mapMissingNote(err error) error {
	return edit.MapMissingNote(err)
}

// Baseline is the state of the note that the caller last read before saving.
type Baseline = edit.Baseline

// Levels updates the index for the named notes, so a write's changes are
// searchable before the write is reported complete.
type Levels = edit.Levels

// Edit applies a change to the contents of an existing note.
//
// The sequence is: read the note, reject content that cannot be read, write the
// identifier, apply the change, store the note, and update the index.
type Edit struct {
	readers vaultport.VaultReaders
	writers vaultport.VaultWriters
	index   Levels
	now     machineport.Clock

	FileStat domain.FileStat
	// ShouldOverwrite writes the caller's text as the whole note: a missing note
	// is created, and no identifier is stamped.
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
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.FileStat, error) {
	return e.shared().Apply(ctx, v, path, change)
}

// shared is the edit as the notes module makes it, over the files of a vault.
func (e Edit) shared() edit.Edit {
	return edit.Edit{
		Open:            e.openFiles,
		Index:           e.index,
		Clock:           e.now,
		FileStat:        e.FileStat,
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

func (f vaultFiles) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return f.reader.StatFile(ctx, path)
}

func (f vaultFiles) Read(ctx context.Context, path string) ([]byte, error) {
	return f.reader.Read(ctx, path)
}

func (f vaultFiles) Write(
	ctx context.Context, path string, content []byte, stat domain.FileStat,
) (domain.FileStat, error) {
	return f.writer.Write(ctx, path, content, stat)
}

func (f vaultFiles) Lock(ctx context.Context) (func(), error) {
	return f.writers.Lock(ctx, f.vault)
}
