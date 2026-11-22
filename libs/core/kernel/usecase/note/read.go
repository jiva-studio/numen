package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Read returns the prose of one note.
type Read struct {
	readers vaultport.VaultReaders
}

func NewRead(readers vaultport.VaultReaders) Read {
	return Read{readers: readers}
}

// Execute reads the note at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome, and the result identifies which one.
func (u Read) Execute(ctx context.Context, v domain.Vault, path string) (edit.Contents, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return edit.Contents{}, err
	}
	return edit.Read(ctx, reader, path)
}
