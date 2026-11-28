package vaultnames

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

type notes struct {
	inner ports.Notes
	names *table
}

// GetNotes returns what the host does with the notes of the vault in its index and its trash, over the paths of the notes.
func (e *Edge) GetNotes(id domain.VaultID, files ports.Files, inner ports.Notes) ports.Notes {
	return &notes{inner: inner, names: e.getTable(id, files)}
}

func (n *notes) EnsureIndexed(ctx context.Context) (host.IndexState, error) {
	return n.inner.EnsureIndexed(ctx)
}

func (n *notes) RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error) {
	if slices.ContainsFunc(paths, func(path string) bool { return !isInvariant(path) && n.names.getName(path) == path }) {
		if err := n.names.refresh(ctx); err != nil {
			return nil, err
		}
	}
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = n.names.getName(path)
	}
	removals, err := n.inner.RemoveNotes(ctx, names)
	for i := range removals {
		removals[i].Path = getNote(removals[i].Path)
	}
	return removals, err
}
