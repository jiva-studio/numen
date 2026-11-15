package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// NoteReader reads the type and the frontmatter of a note from its current file.
type NoteReader interface {
	Execute(ctx context.Context, v domain.Vault, path string) (domain.NoteType, map[string]any, error)
}
