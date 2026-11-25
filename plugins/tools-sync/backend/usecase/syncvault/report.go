package syncvault

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	hostdomain "github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keyConflicts      = "conflicts"
	keyUnverified     = "unverified-changes"
	keyCaseClash      = "case-clash"
	keyStorageFull    = "storage-full"
	keyWentBack       = "restore-went-back"
	keyEpochUntrusted = "restore-epoch-untrusted"
	keyDiverged       = "restore-diverged"

	keyOtherIdentity = "other-identity"

	textWentBack       = "The sync server went back to an earlier state"
	textEpochUntrusted = "The restore announcement of the sync server is not signed by a known root key and was ignored"
	textDiverged       = "The sync server holds another history than this computer"
	textEpochMismatch  = "The sync server reported a restore that its signed announcement does not confirm, so nothing was changed"
	textOtherIdentity  = "This vault was synced with another identity, so it is synced again with the current one."
)

type reporter struct {
	health       ports.Health
	subscription SubscriptionReader
}

func getKey(key string, id hostdomain.VaultID) string { return healthkey.Create(key, string(id)) }

func (r reporter) raise(ctx context.Context, key, text string) error {
	return r.health.Report(ctx, health.Item{Key: key, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}

func (r reporter) settle(ctx context.Context, id hostdomain.VaultID, err error) error {
	var quota syncprotocol.QuotaExceededError
	var notReady syncprotocol.BlobNotReadyError
	switch {
	case errors.As(err, &quota):
		return r.raise(ctx, getKey(keyStorageFull, id), "Cloud storage is full: changes stay on this device")
	case errors.Is(err, syncprotocol.ErrWriteNotAllowed):
		return r.raise(ctx, domain.WarningKey, r.getRefusedText(ctx))
	case errors.As(err, &notReady):
		return nil
	}
	return err
}

// tally is what a pass found to warn of.
type tally struct {
	conflicts  int
	unverified int
	clashes    int
}

func (r reporter) report(ctx context.Context, id hostdomain.VaultID, found tally) error {
	if err := r.health.Clear(ctx, domain.WarningKey); err != nil {
		return err
	}
	if err := r.health.Clear(ctx, getKey(keyStorageFull, id)); err != nil {
		return err
	}
	if err := r.health.Clear(ctx, getKey(keyOtherIdentity, id)); err != nil {
		return err
	}
	if err := r.toggle(ctx, getKey(keyConflicts, id), found.conflicts, GetConflictText(found.conflicts)); err != nil {
		return err
	}
	if err := r.toggle(ctx, getKey(keyCaseClash, id), found.clashes, GetCaseClashText(found.clashes)); err != nil {
		return err
	}
	return r.toggle(ctx, getKey(keyUnverified, id), found.unverified, "Some changes of your other computers could not be verified")
}

func (r reporter) toggle(ctx context.Context, key string, count int, text string) error {
	if count == 0 {
		return r.health.Clear(ctx, key)
	}
	return r.raise(ctx, key, text)
}

// GetConflictText is what the warning of notes changed on two computers says.
func GetConflictText(count int) string {
	if count == 1 {
		return "1 note changed on two computers"
	}
	return fmt.Sprintf("%d notes changed on two computers", count)
}

// GetCaseClashText is what the warning of files whose names match other names says.
func GetCaseClashText(count int) string {
	if count == 1 {
		return "1 file is not synced: its name matches another name"
	}
	return fmt.Sprintf("%d files are not synced: their names match other names", count)
}

func (r reporter) getRefusedText(ctx context.Context) string {
	state, err := r.subscription.Execute(ctx)
	if err == nil {
		if text := domain.GetWarningText(state); text != "" {
			return text
		}
	}
	return domain.GetWarningText(domain.State{Phase: domain.PhaseNone, PeriodEnd: time.Time{}, WritesEnd: time.Time{}, DeleteAt: time.Time{}, PlanID: "", QuotaBytes: 0, UsedBytes: 0, IsTrial: false})
}

func (r reporter) clearRestore(ctx context.Context, id hostdomain.VaultID) error {
	for _, key := range []string{keyWentBack, keyEpochUntrusted, keyDiverged} {
		if err := r.health.Clear(ctx, getKey(key, id)); err != nil {
			return err
		}
	}
	return nil
}
