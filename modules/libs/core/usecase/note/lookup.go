package note

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// TitleQueries is the one question a lookup of titles asks of the index.
type TitleQueries interface {
	// GetNotes returns what is needed to show a note, for the paths asked about.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

// TitlesFound is what the notes at some paths are called. Missing is every path
// that names no note, in the order it was asked for.
type TitlesFound struct {
	Notes   []domain.NoteRef
	Missing []string
}

// GetTitles looks up what the notes at some paths are called.
type GetTitles struct {
	queries TitleQueries
}

func NewGetTitles(queries TitleQueries) GetTitles {
	return GetTitles{queries: queries}
}

// Execute answers for each path, in the order given. A path that names nothing
// is missing and is not an error.
func (u GetTitles) Execute(ctx context.Context, v domain.Vault, paths []string) (TitlesFound, error) {
	found, err := u.queries.GetNotes(ctx, v.ID, paths)
	if err != nil {
		return TitlesFound{}, err
	}
	res := TitlesFound{Notes: make([]domain.NoteRef, 0, len(found))}
	for _, path := range paths {
		if ref, ok := found[path]; ok {
			res.Notes = append(res.Notes, ref)
			continue
		}
		res.Missing = append(res.Missing, path)
	}
	return res, nil
}

// GetNamedPaths finds every note a link written by one name reaches.
type GetNamedPaths struct {
	queries NameQueries
}

func NewGetNamedPaths(queries NameQueries) GetNamedPaths {
	return GetNamedPaths{queries: queries}
}

// Execute returns the paths of the notes filed under the name. More than one
// means a link by that name is ambiguous, and none means no note answers to it.
func (u GetNamedPaths) Execute(ctx context.Context, v domain.Vault, name string) ([]string, error) {
	return u.queries.GetNamedPaths(ctx, v.ID, domain.LinkName(name))
}
