package syncvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keySharedConflicts = "shared-conflicts:"
	keyShareEnded      = "share-ended:"
	textShareEnded     = "The owner ended this share"
)

func (u *SyncVault) executeReceived(ctx context.Context, id domain.VaultID) error {
	result, err := u.steps.Receive.Execute(ctx, id)
	if errors.Is(err, share.ErrEnded) {
		return errors.Join(err, u.reports.raiseEnded(ctx, id))
	}
	if err != nil {
		return err
	}
	if err := u.reports.reportShared(ctx, id, len(result.Conflicts), result.Unverified); err != nil {
		return err
	}
	return u.deps.Progress.WriteStatus(ctx, id, ports.Status{LastSyncedAt: u.deps.Clock.Now(), ConflictCount: len(result.Conflicts)})
}

func (r reporter) raiseEnded(ctx context.Context, id domain.VaultID) error {
	return r.raise(ctx, keyShareEnded+string(id), textShareEnded)
}

func (r reporter) reportShared(ctx context.Context, id domain.VaultID, conflicts, unverified int) error {
	if err := r.toggle(ctx, keySharedConflicts+string(id), conflicts, getSharedConflictText(conflicts)); err != nil {
		return err
	}
	return r.toggle(ctx, keyUnverified+":"+string(id), unverified, "Some changes of this shared vault could not be verified")
}

func getSharedConflictText(count int) string {
	if count == 1 {
		return "1 note of this shared vault was changed here"
	}
	return fmt.Sprintf("%d notes of this shared vault were changed here", count)
}
