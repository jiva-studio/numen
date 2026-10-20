package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetHeadings finds what each of the notes at some paths is divided into.
type GetHeadings struct {
	queries port.NoteQueries
}

func NewGetHeadings(queries port.NoteQueries) *GetHeadings {
	return &GetHeadings{queries: queries}
}

// Execute answers by path, in the order the headings stand in the note.
func (u *GetHeadings) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string][]domain.Heading, error) {
	return u.queries.GetHeadings(ctx, v.ID, paths)
}
