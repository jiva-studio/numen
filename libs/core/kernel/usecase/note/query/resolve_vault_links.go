package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ErrUnknownVault is returned for a vault that is not on the list of this installation.
var ErrUnknownVault = errors.New("no vault")

// ResolveVaultLinks finds where addresses written in one note of a registered vault land.
type ResolveVaultLinks struct {
	vaults  vaultport.VaultList
	resolve *ResolveLinks
}

// NewResolveVaultLinks returns the use case over the list of vaults and the resolution of links.
func NewResolveVaultLinks(vaults vaultport.VaultList, resolve *ResolveLinks) *ResolveVaultLinks {
	return &ResolveVaultLinks{vaults: vaults, resolve: resolve}
}

// Execute resolves each address as written in the vault, or fails with ErrUnknownVault.
func (u *ResolveVaultLinks) Execute(
	ctx context.Context, vaultID domain.VaultID, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	listed, err := u.vaults.List()
	if err != nil {
		return nil, err
	}
	for _, v := range listed {
		if v.ID == vaultID {
			return u.resolve.Execute(ctx, v.ID, from, targets)
		}
	}
	return nil, fmt.Errorf("%w %q", ErrUnknownVault, vaultID)
}
