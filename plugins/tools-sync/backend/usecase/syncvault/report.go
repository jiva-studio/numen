package syncvault

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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

func getKey(key string, id domain.VaultID) string { return key + "/" + string(id) }

func (r reporter) raise(ctx context.Context, id domain.VaultID, key, text string) error {
	return r.health.Report(ctx, health.Item{Key: getKey(key, id), Severity: health.Warning, Text: text, Detail: "", Action: nil})
}

func (r reporter) settle(ctx context.Context, id domain.VaultID, err error) error {
	var quota syncprotocol.QuotaExceededError
	var notReady syncprotocol.BlobNotReadyError
	switch {
	case errors.As(err, &quota):
		return r.raise(ctx, id, keyStorageFull, "Sync storage is full")
	case errors.Is(err, syncprotocol.ErrWriteNotAllowed):
		return r.raise(ctx, id, keySubscription, "Sync needs a subscription")
	case errors.As(err, &notReady):
		return nil
	}
	return err
}

func (r reporter) report(ctx context.Context, id domain.VaultID, conflicts, unverified int) error {
	for _, key := range []string{keyStorageFull, keySubscription} {
		if err := r.health.Clear(ctx, getKey(key, id)); err != nil {
			return err
		}
	}
	if err := r.toggle(ctx, id, keyConflicts, conflicts, GetConflictText(conflicts)); err != nil {
		return err
	}
	return r.toggle(ctx, id, keyUnverified, unverified, "Some changes of your other computers could not be verified")
}

func (r reporter) toggle(ctx context.Context, id domain.VaultID, key string, count int, text string) error {
	if count == 0 {
		return r.health.Clear(ctx, getKey(key, id))
	}
	return r.raise(ctx, id, key, text)
}

// GetConflictText is what the warning of notes changed on two computers says.
func GetConflictText(count int) string {
	if count == 1 {
		return "1 note changed on two computers"
	}
	return fmt.Sprintf("%d notes changed on two computers", count)
}
