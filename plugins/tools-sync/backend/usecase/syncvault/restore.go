package syncvault

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

func (p *pass) pullOrRecover(ctx context.Context) (bool, error) {
	err := p.pull(ctx)
	switch {
	case errors.Is(err, syncprotocol.ErrNoVault):
		return p.recover(ctx)
	case err != nil:
		return false, err
	case p.verdict == restore.WentBack:
		return false, p.use.reports.raise(ctx, keyWentBack, textWentBack)
	case p.verdict == restore.Changed:
		return p.recover(ctx)
	}
	return true, p.use.reports.health.Clear(ctx, keyWentBack)
}

func (p *pass) recover(ctx context.Context) (bool, error) {
	err := p.use.steps.RecoverAccount.Execute(ctx)
	if err == nil {
		err = p.use.steps.RecoverVault.Execute(ctx, p.id)
	}
	switch {
	case errors.Is(err, restore.ErrKeyChanged):
		return false, p.use.reports.raise(ctx, keyKeyChanged, textKeyChanged)
	case errors.Is(err, restore.ErrDiverged):
		return false, p.use.reports.raise(ctx, keyDiverged, textDiverged)
	case err != nil:
		return false, p.use.reports.settle(ctx, err)
	}
	if err := p.use.reports.clearRestore(ctx); err != nil {
		return false, err
	}
	if err := p.pull(ctx); err != nil {
		return false, err
	}
	return p.verdict == restore.Same, nil
}

// pinRestoreKey pins the key that signs restore epochs the first time a pass sees one, before the server has announced
// any epoch, so a server that later signs an epoch with another key is refused as a changed key.
func (p *pass) pinRestoreKey(ctx context.Context) error {
	deps := p.use.deps.Restore
	pinned, err := deps.Mirror.ReadPin(ctx, deps.Address)
	if err != nil || len(pinned) > 0 {
		return err
	}
	_, key, err := deps.Epochs.GetRestoreEpoch(ctx)
	if err != nil || len(key) == 0 {
		return err
	}
	return deps.Mirror.WritePin(ctx, deps.Address, key)
}
