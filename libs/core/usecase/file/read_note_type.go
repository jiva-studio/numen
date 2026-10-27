package file

import (
	"context"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// IsNote reports whether the file at the path is a note, by its extension.
func IsNote(path string) bool {
	return strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension)
}

// ReadNoteType is the type the note at the path carries in its frontmatter as the file now reads,
// so a note just written has its type before the index holds it. A file that is no note has no
// type and is not read.
func ReadNoteType(ctx context.Context, readers vaultport.VaultReaders, v domain.Vault, path string) (domain.NoteType, error) {
	if !IsNote(path) {
		return "", nil
	}
	reader, err := readers.Open(v)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	return getNoteType(raw), nil
}

// getNoteType is the type a note's bytes carry.
func getNoteType(raw []byte) domain.NoteType {
	return markdown.Parse(domain.FileStat{}, raw).Type
}
