package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Titles is what the notes at some paths are called. Missing is every path that
// names no note, in the order it was asked for.
type Titles struct {
	Notes   []domain.NoteRef
	Missing []string
}

// GetTitles looks up what the notes at some paths are called.
type GetTitles struct {
	queries port.NoteQueries
}

func NewGetTitles(queries port.NoteQueries) GetTitles {
	return GetTitles{queries: queries}
}

// Execute answers for each path, in the order given. A path that names nothing
// is missing and is not an error.
func (u GetTitles) Execute(ctx context.Context, v domain.Vault, paths []string) (Titles, error) {
	found, err := u.queries.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return Titles{}, err
	}
	res := Titles{Notes: make([]domain.NoteRef, 0, len(found))}
	for _, path := range paths {
		if ref, ok := found[path]; ok {
			res.Notes = append(res.Notes, ref)
			continue
		}
		res.Missing = append(res.Missing, path)
	}
	return res, nil
}
