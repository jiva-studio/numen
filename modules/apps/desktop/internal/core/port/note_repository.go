package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// NoteRepository holds notes. Every method takes a vault: one database holds
// them all, and a call that forgets its vault silently touches another vault's
// notes.
type NoteRepository interface {
	// Save puts a group of notes in at once. A group rather than a note
	// because the cost of storing one is dominated by the boundary around it
	// rather than by the note, and because a note and the fingerprint that says
	// it is up to date have to become true together — an index holding a note
	// it believes current, when it never finished writing it, is worse than an
	// index missing it.
	Save(ctx context.Context, vaultID string, notes []domain.Note) error
	// Remove takes out the notes whose files are gone. The vault is
	// authoritative: what is not on disk is not in the index.
	Remove(ctx context.Context, vaultID string, paths []string) error
}
