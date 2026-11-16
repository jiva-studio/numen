package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// FrontmatterWrite is the keys to set on one note. A nil value removes the key.
type FrontmatterWrite struct {
	Path     string
	FileStat domain.FileStat
	Values   map[string]any
}

// FrontmatterWritten is the outcome of one write. Error is nil when the note was written, and FileStat
// is then the note's current state.
type FrontmatterWritten struct {
	FileStat domain.FileStat
	Error    error
}

// WriteFrontmatter sets, replaces and removes top-level frontmatter keys of notes.
type WriteFrontmatter struct {
	readers vaultport.VaultReaders
	writers NoteWriting
	index   edit.IndexUpdater
	now     machineport.Clock
}

// NewWriteFrontmatter writes frontmatter keys through the vault readers and writers, the index updater and the clock.
func NewWriteFrontmatter(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock,
) WriteFrontmatter {
	return WriteFrontmatter{readers: readers, writers: writers, index: index, now: now}
}

// Execute writes the keys of each note and answers for each one, in the order given.
func (u WriteFrontmatter) Execute(ctx context.Context, v domain.Vault, writes []FrontmatterWrite) ([]FrontmatterWritten, error) {
	results := make([]FrontmatterWritten, len(writes))
	for i := range results {
		results[i].Error = errors.New("not implemented")
	}
	return results, nil
}
