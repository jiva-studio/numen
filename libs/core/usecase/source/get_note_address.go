package source

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GetNoteAddress says what address the note at a path points at, and nothing where the file is not a link note.
type GetNoteAddress struct {
	texts sourceTexts
}

func (u GetNoteAddress) Execute(ctx context.Context, v domain.Vault, path string) domain.URL {
	return u.texts.GetNoteURL(ctx, v, path)
}
