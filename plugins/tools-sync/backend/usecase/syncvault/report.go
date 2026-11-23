package syncvault

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	domain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/subscription"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keyConflicts   = "conflicts"
	keyUnverified  = "unverified-changes"
	keyStorageFull = "storage-full"
)

type reporter struct {
	health       ports.Health
	subscription SubscriptionReader
}

func (r reporter) raise(ctx context.Context, key, text string) error {
	return r.health.Report(ctx, health.Item{Key: key, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}

func (r reporter) settle(ctx context.Context, err error) error {
	var quota syncprotocol.QuotaExceededError
	var notReady syncprotocol.BlobNotReadyError
	switch {
	case errors.As(err, &quota):
		return r.raise(ctx, keyStorageFull, "Cloud storage is full: changes stay on this device")
	case errors.Is(err, syncprotocol.ErrWriteNotAllowed):
		return r.raise(ctx, domain.WarningKey, r.getRefusedText(ctx))
	case errors.As(err, &notReady):
		return nil
	}
	return err
}

func (r reporter) report(ctx context.Context, conflicts, previousConflicts, unverified int) error {
	for _, key := range []string{keyStorageFull, domain.WarningKey} {
		if err := r.health.Clear(ctx, key); err != nil {
			return err
		}
	}
	if err := r.toggle(ctx, keyConflicts, conflicts > 0 || previousConflicts > 0, conflicts, GetConflictText(conflicts)); err != nil {
		return err
	}
	return r.toggle(ctx, keyUnverified, unverified > 0, unverified, "Some changes of your other computers could not be verified")
}

func (r reporter) toggle(ctx context.Context, key string, isOn bool, count int, text string) error {
	if count == 0 {
		if isOn {
			return r.health.Clear(ctx, key)
		}
		return nil
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

func (r reporter) getRefusedText(ctx context.Context) string {
	state, err := r.subscription.Execute(ctx)
	if err == nil {
		if text := domain.GetWarningText(state); text != "" {
			return text
		}
	}
	return domain.GetWarningText(domain.State{Phase: domain.PhaseNone, PeriodEnd: time.Time{}, WritesEnd: time.Time{}, DeleteAt: time.Time{}})
}
