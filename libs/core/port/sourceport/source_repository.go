package sourceport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// SourceRepository holds the sources of one vault. Every method takes a vault:
// one database holds them all, and a call that forgets its vault touches another
// vault's sources.
type SourceRepository interface {
	// SaveSource records the current state of a file. A source saved without a
	// recipe needs its text produced: the file has changed or has not been read
	// yet.
	SaveSource(ctx context.Context, vaultID domain.VaultID, s domain.Source) error

	// SaveExtraction records a source and replaces its chunks with the ones its
	// text was cut into. Both arrive in one write, so a recipe is never
	// recorded for chunks that are not there.
	SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error

	// RemoveSources deletes the sources of one kind at the given paths, and
	// everything derived from them. A source the vault does not hold cannot be
	// read, so a passage naming it can never be displayed; a row left in the
	// index would match a search but could not be opened.
	RemoveSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error

	// MoveSources moves what the vault holds at one path to another path, with
	// everything under it. The path of a source belongs to this port regardless
	// of file kind, so a folder of notes and books moves in one write.
	//
	// A note without an explicit name takes the filename it moves to as its
	// name, when that filename is the name its own title is filed under. A note
	// with an explicit name that differs from the filename keeps that name.
	MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error
}
