package syncvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	keyConflicts    = "conflicts"
	keyUnverified   = "unverified-changes"
	keyStorageFull  = "storage-full"
	keySubscription = "subscription-needed"
)

type reporter struct{ health ports.Health }

func (r reporter) raise(ctx context.Context, key, text string) error {
	return r.health.Report(ctx, health.Item{Key: key, Severity: health.Warning, Text: text, Detail: "", Action: nil})
}

func (r reporter) settle(ctx context.Context, err error) error {
	var quota syncprotocol.QuotaExceededError
	var notReady syncprotocol.BlobNotReadyError
	switch {
	case errors.As(err, &quota):
		return r.raise(ctx, keyStorageFull, "Sync storage is full")
	case errors.Is(err, syncprotocol.ErrWriteNotAllowed):
		return r.raise(ctx, keySubscription, "Sync needs a subscription")
	case errors.As(err, &notReady):
		return nil
	}
	return err
}

// report raises the warnings of a pass that went through and clears the ones it no longer needs.
func (r reporter) report(ctx context.Context, conflicts, previousConflicts, unverified int) error {
	for _, key := range []string{keyStorageFull, keySubscription} {
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
