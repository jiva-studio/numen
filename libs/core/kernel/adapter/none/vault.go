package none

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrNotOpen is what VaultReaders answers for every vault.
var ErrNotOpen = errors.New("this vault's files are not open")

// VaultReaders opens no vault.
type VaultReaders struct{}

func (VaultReaders) Open(domain.Vault) (vaultport.VaultReader, error) { return nil, ErrNotOpen }

// IndexClaim holds nothing and is never busy.
type IndexClaim struct{}

func (IndexClaim) ClaimIndex(context.Context) (func() error, error) {
	return func() error { return nil }, nil
}
