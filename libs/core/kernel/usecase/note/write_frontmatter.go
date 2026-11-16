package note

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
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

// NewWriteFrontmatter returns the use case that writes frontmatter keys. It takes the vault readers
// and writers, the index updater and the clock, as NewEditLinks does.
func NewWriteFrontmatter(
	readers vaultport.VaultReaders, writers NoteWriting, index edit.IndexUpdater, now machineport.Clock,
) WriteFrontmatter {
	return WriteFrontmatter{readers: readers, writers: writers, index: index, now: now}
}

// Execute writes the keys of each note and answers for each one, in the order given. A note that
// cannot be written costs the others nothing. When ctx is canceled, every note not yet written gets
// plugin.ErrNotAttempted, and the results are returned with a nil error.
func (u WriteFrontmatter) Execute(ctx context.Context, v domain.Vault, writes []FrontmatterWrite) ([]FrontmatterWritten, error) {
	results := make([]FrontmatterWritten, len(writes))
	for i, one := range writes {
		var stat domain.FileStat
		err := ctx.Err()
		if err == nil {
			e := NewEdit(u.readers, u.writers, u.index, u.now)
			e.FileStat = one.FileStat
			stat, err = e.Apply(ctx, v, one.Path, func(doc *markdown.Document) error {
				return setKeys(doc, one.Values)
			})
		}
		if err != nil && ctx.Err() != nil && !errors.Is(err, edit.ErrIndexOutdated) {
			err = fmt.Errorf("the call was canceled before the frontmatter of %s was written: %w", one.Path, plugin.ErrNotAttempted)
		}
		results[i].Error = err
		if err == nil {
			results[i].FileStat = stat
		}
	}
	return results, nil
}

func setKeys(doc *markdown.Document, values map[string]any) error {
	for _, key := range slices.Sorted(maps.Keys(values)) {
		var err error
		if value := values[key]; value == nil {
			err = doc.Remove(key)
		} else if names, isList := asNames(value); isList {
			err = doc.SetList(key, names)
		} else {
			err = doc.SetValue(key, value)
		}
		if err != nil {
			return fmt.Errorf("write %q: %w", key, err)
		}
	}
	return nil
}

func asNames(value any) ([]string, bool) {
	switch list := value.(type) {
	case []string:
		return list, true
	case []any:
		names := make([]string, len(list))
		for i, item := range list {
			name, isText := item.(string)
			if !isText {
				return nil, false
			}
			names[i] = name
		}
		return names, true
	}
	return nil, false
}
