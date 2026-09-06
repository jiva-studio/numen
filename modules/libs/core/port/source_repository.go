package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// SourceRepository holds the sources of one vault. Every method takes a vault:
// one database holds them all, and a call that forgets its vault touches another
// vault's sources.
type SourceRepository interface {
	// SaveSource records what a file is now. A source saved with no recipe owes
	// its text: the file it names has changed, or nothing has read it yet.
	SaveSource(ctx context.Context, vaultID domain.VaultID, s domain.Source) error

	// SaveExtraction records a source and replaces its chunks with the ones its
	// text was cut into. Both arrive in one write, so a recipe is never
	// recorded for chunks that are not there.
	SaveExtraction(ctx context.Context, vaultID domain.VaultID, e domain.SourceChunks) error

	// RemoveSources takes out the sources of one kind at the paths given, and
	// everything derived from them. A source the vault no longer holds cannot be
	// read, so a passage naming it can never be shown: leaving it in the index
	// leaves a row that answers a search and then cannot be looked at.
	RemoveSources(ctx context.Context, vaultID domain.VaultID, kind domain.SourceKind, paths []string) error

	// MoveSources files what the vault held at one path under another, with
	// everything under it. The path of a source is this port's, whatever kind of
	// file it is, so a folder of notes and books travels in one write.
	//
	// A note is called by the filename it lands under where nothing inside the
	// file names it, and where the new filename is the name its own title is
	// filed under. A note carrying a name the filename is not takes that name
	// with it.
	MoveSources(ctx context.Context, vaultID domain.VaultID, from, to string) error
}
