package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// SourceQueries answers what the index holds about sources, in shapes that are
// not sources: what it believes each file was, and which of them owe their text.
//
// A limit bounds every answer that is a list, so that the caller works through a
// library of any size in pieces.
type SourceQueries interface {
	// Fingerprints is what the index believes about each file of one kind, keyed
	// by path, so a walk can decide what to read without opening anything.
	Fingerprints(ctx context.Context, vaultID string, kind domain.SourceKind) (map[string]domain.FileRef, error)

	// Unchunked is the sources of one kind with no small window: the file
	// changed, or nothing has cut it yet.
	Unchunked(ctx context.Context, vaultID string, kind domain.SourceKind, limit int) ([]string, error)

	// ByOtherRecipe is the sources of one kind whose text was not produced by
	// the recipe given. Their chunks describe text the extractor in use would
	// not produce.
	ByOtherRecipe(ctx context.Context, vaultID string, kind domain.SourceKind, recipe string, limit int) ([]string, error)
}
