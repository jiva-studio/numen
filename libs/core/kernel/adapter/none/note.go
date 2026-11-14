package none

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// NoteQueries is the index of a build that has none: it holds no vault, no note and no name.
type NoteQueries struct{}

func (NoteQueries) GetNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, nil
}

func (NoteQueries) GetNotes(context.Context, domain.VaultID, []string) (map[string]domain.NoteRef, error) {
	return nil, nil
}

func (NoteQueries) Has(context.Context, domain.VaultID) (bool, error) { return false, nil }

func (NoteQueries) GetSummary(context.Context, domain.VaultID) (domain.VaultSummary, error) {
	return domain.VaultSummary{}, nil
}
