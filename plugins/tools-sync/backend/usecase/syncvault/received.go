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

// executeReceived brings the changes of the owner into a received vault. It pushes nothing and asks the server for no
// write. A share the owner ended is reported and returned as share.ErrEnded.
func (u *SyncVault) executeReceived(ctx context.Context, id domain.VaultID) error {
	previous, err := u.deps.Progress.ReadStatus(ctx, id)
	if err != nil {
		return err
	}
	result, err := u.steps.Receive.Execute(ctx, id)
	if errors.Is(err, share.ErrEnded) {
		return errors.Join(err, u.reports.raiseEnded(ctx, id))
	}
	if err != nil {
		return err
	}
	if err := u.reports.reportShared(ctx, id, len(result.Conflicts), previous.ConflictCount, result.Unverified); err != nil {
		return err
	}
	return u.deps.Progress.WriteStatus(ctx, id, ports.Status{LastSyncedAt: u.deps.Clock.Now(), ConflictCount: len(result.Conflicts)})
}

func (r reporter) raiseEnded(ctx context.Context, id domain.VaultID) error {
	return r.raise(ctx, keyShareEnded+string(id), textShareEnded)
}

func (r reporter) reportShared(ctx context.Context, id domain.VaultID, conflicts, previousConflicts, unverified int) error {
	if err := r.toggle(ctx, keySharedConflicts+string(id), conflicts > 0 || previousConflicts > 0, conflicts, getSharedConflictText(conflicts)); err != nil {
		return err
	}
	return r.toggle(ctx, keyUnverified+":"+string(id), unverified > 0, unverified, "Some changes of this shared vault could not be verified")
}

// getSharedConflictText is what the warning of notes of a shared vault that the reader changed says.
func getSharedConflictText(count int) string {
	if count == 1 {
		return "1 note of this shared vault was changed here"
	}
	return fmt.Sprintf("%d notes of this shared vault were changed here", count)
}
