package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// UserSwitch shows a vault as a person asks for it, through the switch it is given. No plugin open is
// running for such a switch.
type UserSwitch struct {
	switching VaultSwitch
}

// NewUserSwitch makes the switch a person shows another vault through.
func NewUserSwitch(switching VaultSwitch) UserSwitch { return UserSwitch{switching: switching} }

// ShowVault shows v in the window.
func (u UserSwitch) ShowVault(ctx context.Context, v domain.Vault) error {
	return u.switching.Execute(ctx, v, nil)
}
