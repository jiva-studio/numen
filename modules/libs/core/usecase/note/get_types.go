package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// GetTypes finds what each of the notes at some paths is.
type GetTypes struct {
	queries port.NoteQueries
}

func NewGetTypes(queries port.NoteQueries) GetTypes {
	return GetTypes{queries: queries}
}

// Execute answers by path. A path naming no note is absent.
func (u GetTypes) Execute(ctx context.Context, v domain.Vault, paths []string) (map[string]domain.NoteType, error) {
	return u.queries.GetTypes(ctx, v.ID, paths)
}
