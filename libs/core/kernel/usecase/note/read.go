package note

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Read returns the prose of one note.
type Read struct {
	readers vaultport.VaultReaders
}

var errNotOpen = errors.New("this vault's files are not open")

func NewRead(readers vaultport.VaultReaders) Read {
	return Read{readers: getReaders(readers)}
}

// Execute reads the note at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome, and the result identifies which one.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (edit.Contents, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return edit.Contents{}, err
	}
	return edit.Read(ctx, reader, path)
}

type ReadFrontmatter struct {
	readers vaultport.VaultReaders
}

func NewReadFrontmatter(readers vaultport.VaultReaders) ReadFrontmatter {
	return ReadFrontmatter{readers: readers}
}

// Execute returns the type and the frontmatter of the note at the path, read from the current file;
// a path that holds no note has neither.
func (u ReadFrontmatter) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, map[string]any, error) {
	if !domain.IsNoteFile(path) {
		return "", map[string]any{}, nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", nil, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", nil, err
	}
	note := markdown.Parse(domain.FileStat{}, raw)
	if note.Frontmatter == nil {
		return note.Type, map[string]any{}, nil
	}
	return note.Type, note.Frontmatter, nil
}
