package edit

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/markdown"
)

// ReadHead returns the type and the frontmatter of the note at path, read from its current file; a path that
// holds no note has neither.
func ReadHead(ctx context.Context, files Reader, path string) (domain.NoteType, map[string]any, error) {
	if !domain.IsNoteFile(path) {
		return "", map[string]any{}, nil
	}
	raw, err := files.Read(ctx, path)
	if err != nil {
		return "", nil, err
	}
	note := markdown.Parse(domain.FileStat{}, raw)
	if note.Frontmatter == nil {
		return note.Type, map[string]any{}, nil
	}
	return note.Type, note.Frontmatter, nil
}
