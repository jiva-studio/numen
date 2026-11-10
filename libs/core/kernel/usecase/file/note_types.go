package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
)

func getNoteTypes(
	ctx context.Context, queries noteport.NoteTypes, v domain.Vault, entries []domain.Entry,
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
