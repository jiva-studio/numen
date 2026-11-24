package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
)

var (
	errNoVault    = errors.New("the call names no vault")
	errOtherVault = errors.New("the call names a vault the window does not show, and the tasks serve only the one it shows")
	errNotReady   = errors.New("the tasks are still being read, try again in a moment")
)

// Vaults gives a tool or a query function the index of the vault the window shows, the one the plugin
// keeps, and refuses a call that names another vault or none.
type Vaults struct {
	getShownVault func(ctx context.Context) (domain.VaultID, error)
	tasks         *index.Index
	scope         tasktools.Scope
}

// NewVaults returns the vaults of the index; shown tells which vault the index holds.
func NewVaults(shown func(ctx context.Context) (domain.VaultID, error), tasks *index.Index, scope tasktools.Scope) *Vaults {
	return &Vaults{getShownVault: shown, tasks: tasks, scope: scope}
}

// GetScope returns the tasks, the editor and the tick of the vault.
func (v *Vaults) GetScope(ctx context.Context, vaultID string) (tasktools.Scope, error) {
	if err := v.check(ctx, vaultID); err != nil {
		return tasktools.Scope{}, err
	}
	return v.scope, nil
}

// GetTasks returns the tasks of the vault, on the same conditions as GetScope.
func (v *Vaults) GetTasks(ctx context.Context, vaultID string) (queryfns.Tasks, error) {
	if err := v.check(ctx, vaultID); err != nil {
		return nil, err
	}
	return v.tasks, nil
}

func (v *Vaults) check(ctx context.Context, vaultID string) error {
	if vaultID == "" {
		return errNoVault
	}
	shown, err := v.getShownVault(ctx)
	if err != nil {
		return convertError("read the shown vault", err)
	}
	if domain.VaultID(vaultID) != shown {
		return errOtherVault
	}
	if !v.tasks.GetState().IsReady {
		return errNotReady
	}
	return nil
}
