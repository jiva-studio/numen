package syncvault

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/restore"
)

// pullOrRecover pulls the changes, and runs the recovery first when the server announced a restore or lost the vault.
// It reports false when the pass must not go on: the server went back to an earlier state, or the recovery stopped.
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
	return !p.isReset, p.use.reports.health.Clear(ctx, keyWentBack)
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
	return p.verdict == restore.Same && !p.isReset, nil
}
