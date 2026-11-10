package note

import (
	"context"
	"crypto/rand"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Edit applies a change to the contents of an existing note.
//
// The sequence is: read the note, reject content that cannot be read, write the
// identifier, apply the change, store the note, and update the index.
type Edit struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	now     machineport.Clock

	FileStat        domain.FileStat
	ShouldOverwrite bool
	Bound           int
	Baseline        *edit.Baseline
}

// NewEdit is what a change to an existing note is made through: the vault it
// is read and written through, what updates the index for it, and what time
// it is, because an edit stamps the identifier a note arrived without. What the
// caller believes is on disk, and how much of a file it will hold, are set
// beside it.
func NewEdit(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock,
) Edit {
	return Edit{readers: readers, writers: writers, index: index, now: now}
}

// Apply makes one change to the note at path and puts it back.
func (e Edit) Apply(ctx context.Context, v domain.Vault, path string, change func(*markdown.Document) error) (domain.FileStat, error) {
	return e.shared().Apply(ctx, v, path, change)
}

func (e Edit) shared() edit.Edit {
	return edit.Edit{
		Open:            e.openFiles,
		Index:           e.index,
		Clock:           e.now,
		Entropy:         rand.Reader,
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
	writer, err := e.writers.OpenNoteWriter(v)
	if err != nil {
		return nil, err
	}
	return vaultFiles{reader: reader, writer: writer, writers: e.writers, vault: v}, nil
}

// vaultFiles are the files of one vault, read and written through the ports.
type vaultFiles struct {
	reader  vaultport.VaultReader
	writer  vaultport.NoteWriter
	writers NoteWriting
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
