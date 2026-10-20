package file

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetFingerprint finds what the vault says about the file at a path: its kind, its size and when it changed.
type GetFingerprint struct {
	readers port.VaultReaders
}

func NewGetFingerprint(readers port.VaultReaders) *GetFingerprint {
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
