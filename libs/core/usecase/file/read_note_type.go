package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// IsNote reports whether the file at the path is a note, by its extension.
func IsNote(path string) bool {
	return vaultport.IsNote(path)
}

// ReadNoteType is the type the note at the path carries in its frontmatter as the file now reads,
// so a note just written has its type before the index holds it. A file that is no note has no
// type and is not read.
func ReadNoteType(ctx context.Context, readers vaultport.VaultReaders, v domain.Vault, path string) (domain.NoteType, error) {
	return vaultport.ReadNoteType(ctx, readers, v, path)
}

// getNoteType is the type a note's bytes carry.
func getNoteType(raw []byte) domain.NoteType {
	return markdown.Parse(domain.FileStat{}, raw).Type
}
