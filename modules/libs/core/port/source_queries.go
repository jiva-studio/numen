package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// SourceQueries answers what the index holds about sources, in shapes that are
// not sources: what it believes each file was, and which of them owe their text.
//
// A limit bounds every answer that is a list, so that the caller works through a
// library of any size in pieces.
type SourceQueries interface {
	// Fingerprints is what the index believes about each file of one kind, keyed
	// by path, so a walk can decide what to read without opening anything.
	Fingerprints(ctx context.Context, vaultID string, kind domain.SourceKind) (map[string]domain.Fingerprint, error)

	// Under is every source the vault holds at a path and beneath it: the one
	// file, or everything a folder holds, by path. The index holds a row per
	// file with the path it is filed under, and one query reads them.
	Under(ctx context.Context, vaultID, path string) ([]domain.Fingerprint, error)

	// Unchunked is the sources of one kind with no small chunk: the file
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

	// Reading is what one source's text came from. It answers false where the
	// index holds no source at that path.
	//
	// Producer is empty for a source whose own bytes are the text, which is the
	// ordinary case, and a caller acts on the difference: it decides which
	// producer the offsets a chunk carries belong to.
	Reading(ctx context.Context, vaultID, path string) (SourceText, bool, error)

	// Recognised is the sources of one kind whose text a producer made, by path.
	//
	// A scan asks it to find the ones whose files are gone: the store is a
	// folder on the person's disk and they may empty it.
	Recognised(ctx context.Context, vaultID string, kind domain.SourceKind) ([]SourceText, error)
}

// SourceText is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that reading are kept under.
type SourceText struct {
	Path     string
	Producer string
	Hash     string

	// Size and ModTime are the file as the index last saw it. A reading is of
	// the bytes that were there then, and a file rewritten since is one those
	// coordinates no longer describe.
	Size    int64
	ModTime int64
}
