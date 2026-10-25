package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
)

// NameQueries is the one question writing a link, or naming a new note, asks
// of the vault.
type NameQueries = edit.NameQueries

// ErrUnaddressable is a note no link reaches: its name carries a character a
// link is read up to. Nothing is written.
var ErrUnaddressable = edit.ErrUnaddressable

// GetAddress is how a note the application knows by path is written into a
// link: by its name, or by its path where the name would mean another note.
func GetAddress(ctx context.Context, names NameQueries, vaultID domain.VaultID, path string) (domain.Address, error) {
	return edit.GetAddress(ctx, names, vaultID, path)
}
