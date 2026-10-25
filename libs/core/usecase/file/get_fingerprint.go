package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// GetFingerprint finds what the vault says about the file at a path: its kind, its size and when it changed.
type GetFingerprint struct {
	readers vaultport.VaultReaders
}

func NewGetFingerprint(readers vaultport.VaultReaders) *GetFingerprint {
	return &GetFingerprint{readers: readers}
}

// Execute reads no bytes of the file.
func (u *GetFingerprint) Execute(ctx context.Context, v domain.Vault, path string) (domain.Fingerprint, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.Fingerprint{}, err
	}
	return reader.GetFingerprint(ctx, path)
}
