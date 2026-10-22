package noteport

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// NoteRepository holds notes. Every method takes a vault: one database holds
// them all, and a call that forgets its vault silently touches another vault's
// notes.
type NoteRepository interface {
	// Save puts a group of notes in at once, and either all of them arrive or
	// none do. Each arrives with the headings and chunks it is found by.
	Save(ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks) error
	// Remove takes out the notes whose files are gone. The vault is
	// authoritative: what is not on disk is not in the index.
	Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error
}
