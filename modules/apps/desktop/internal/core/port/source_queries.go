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
	// any of the recipes given. Their chunks describe text no reader in use
	// would produce now.
	//
	// There is a recipe for every reader, because a vault holds files of more
	// than one format and what took the text out is part of what produced the
	// offsets. Asked with one recipe, this would name every source of every
	// other format on every run.
	ByOtherRecipe(ctx context.Context, vaultID string, kind domain.SourceKind, recipes []string, limit int) ([]string, error)

	// Recognised is the sources of one kind whose text a producer made, by path.
	//
	// A scan asks it to find the ones whose files are gone: the store is a
	// folder on the person's disk and they may empty it.
	Recognised(ctx context.Context, vaultID string, kind domain.SourceKind) ([]Recognised, error)
}

// Recognised is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that reading are kept under.
type Recognised struct {
	Path string
	From string
	Hash string
}
