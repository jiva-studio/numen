package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// NoteQueries answers questions about notes in shapes that are not notes.
type NoteQueries interface {
	// Fingerprints is what the index believes about each file, keyed by path, so
	// a scan can decide what to reparse without reading anything.
	Fingerprints(ctx context.Context, vaultID string) (map[string]domain.FileRef, error)
	Search(ctx context.Context, vaultID, query string, limit int) ([]domain.NoteMatch, error)
	Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error)
}
