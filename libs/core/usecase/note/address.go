package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
)

// NameQueries is the single query that writing a link, or naming a new note,
// makes of the vault.
type NameQueries = edit.NameQueries

// ErrUnaddressable means no link can address the note: its name contains a
// character at which link parsing stops. Nothing is written.
var ErrUnaddressable = edit.ErrUnaddressable

// GetAddress returns the form in which a note known by path is written into a
// link: by its name, or by its path when the name would match another note.
func GetAddress(ctx context.Context, names NameQueries, vaultID domain.VaultID, path string) (domain.Address, error) {
	return edit.GetAddress(ctx, names, vaultID, path)
}
