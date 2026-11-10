package startup

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// SwitchVault shows a vault in the window and, where it was shown, points the plugins at it.
type SwitchVault[Open any] struct {
	Show        func(ctx context.Context, v domain.Vault, open Open) error
	FollowVault func(path string)
	After       func(v domain.Vault, err error)
}

// Execute returns the error of showing the vault.
func (u SwitchVault[Open]) Execute(ctx context.Context, v domain.Vault, open Open) error {
	err := u.Show(ctx, v, open)
	if err == nil {
		u.FollowVault(v.Path)
	}
	u.After(v, err)
	return err
}
