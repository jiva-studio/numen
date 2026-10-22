package file

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/noteport"
)

// getNoteTypes is what each note among the entries is, keyed by path, and nothing when there is no index.
func getNoteTypes(
	ctx context.Context, queries noteport.NoteQueries, v domain.Vault, entries []domain.Entry,
) (map[string]domain.NoteType, error) {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsFolder && entry.Kind == domain.KindNote {
			paths = append(paths, entry.Path)
		}
	}
	if queries == nil || len(paths) == 0 {
		return nil, nil
	}
	return queries.GetTypes(ctx, v.ID, paths)
}
