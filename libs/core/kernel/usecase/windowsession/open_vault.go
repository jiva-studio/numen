package windowsession

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// OpenVault shows the vault a person picks from the list in the window, through the switch it is given. No
// plugin open is running for such a switch.
type OpenVault struct {
	lookup    vaultport.VaultLookup
	switching VaultSwitch
}

// NewOpenVault makes the opening over the list this installation keeps and the switch of the window.
func NewOpenVault(lookup vaultport.VaultLookup, switching VaultSwitch) *OpenVault {
	return &OpenVault{lookup: lookup, switching: switching}
}

// Execute shows the vault with the id.
func (u *OpenVault) Execute(ctx context.Context, id string) error {
	v, err := vaultdomain.FindByName(id, u.lookup.FindVault)
	if err != nil {
		return err
	}
	return u.switching.Execute(ctx, v, nil)
}
