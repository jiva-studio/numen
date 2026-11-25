package syncvault

import (
	"bytes"
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
		return false, p.use.reports.raise(ctx, getKey(keyWentBack, p.id), textWentBack)
	case p.verdict == restore.Changed:
		return p.recoverIfAnnounced(ctx)
	}
	return true, p.use.reports.clearRestore(ctx, p.id)
}

func (p *pass) recoverIfAnnounced(ctx context.Context) (bool, error) {
	epoch, _, err := p.use.deps.Restore.Epochs.GetRestoreEpoch(ctx)
	if err != nil || (epoch != nil && bytes.Equal(epoch.GetId(), p.epochID)) {
		return p.recover(ctx)
	}
	return false, p.use.reports.raise(ctx, getKey(keyEpochUntrusted, p.id), textEpochMismatch)
}

func (p *pass) recover(ctx context.Context) (bool, error) {
	err := p.use.steps.RecoverAccount.Execute(ctx)
	if err == nil {
		err = p.use.steps.RecoverVault.Execute(ctx, p.id)
	}
	if errors.Is(err, restore.ErrForeignJournal) {
		return p.registerAgain(ctx)
	}
	switch {
	case errors.Is(err, restore.ErrEpochUntrusted):
		return false, p.use.reports.raise(ctx, getKey(keyEpochUntrusted, p.id), textEpochUntrusted)
	case errors.Is(err, restore.ErrDiverged):
		return false, p.use.reports.raise(ctx, getKey(keyDiverged, p.id), textDiverged)
	case err != nil:
		return false, p.use.reports.settle(ctx, p.id, err)
	}
	if err := p.use.reports.clearRestore(ctx, p.id); err != nil {
		return false, err
	}
	if err := p.pull(ctx); err != nil {
		return false, err
	}
	return p.verdict == restore.Same, nil
}

func (p *pass) registerAgain(ctx context.Context) (bool, error) {
	if err := p.use.deps.Forget.ForgetVault(ctx, p.id); err != nil {
		return false, err
	}
	if err := p.use.steps.Register.Execute(ctx, p.id); err != nil {
		return false, p.use.reports.settle(ctx, p.id, err)
	}
	if err := p.pull(ctx); err != nil {
		return false, err
	}
	return p.verdict == restore.Same, nil
}
