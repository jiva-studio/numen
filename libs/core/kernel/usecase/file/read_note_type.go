package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ReadNoteType returns the type the note at a path carries in its frontmatter.
type ReadNoteType struct {
	readers vaultport.VaultReaders
}

func NewReadNoteType(readers vaultport.VaultReaders) ReadNoteType {
	return ReadNoteType{readers: readers}
}

// Execute returns the type the note at the path carries in its frontmatter as the file currently reads,
// so a note created in this session has its type before the index holds it. A file that is not a note has no
// type and is not read.
func (u ReadNoteType) Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, error) {
	if !domain.IsNoteFile(path) {
		return "", nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	return markdown.Parse(domain.FileStat{}, raw).Type, nil
}
