package service

import (
	"context"
	"log/slog"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/adapter/cloud"
)

const keyEpochRoot = "tools-sync/epoch-root"

func (r *runtime) readEpochRoots(ctx context.Context, address string) cloud.Roots {
	roots, err := cloud.GetRoots(address, r.epochRoot.Get())
	if r.parts.health == nil {
		return roots
	}
	if err != nil {
		reportErr := r.parts.health.Report(ctx, health.Item{
			Key: keyEpochRoot, Severity: health.Warning, Text: "The epoch root key of Numen Sync is not 32 bytes of hex",
			Detail: "Restore announcements of the sync server are ignored until the key is fixed.", Action: health.OpenSettings(settingEpochRoot),
		})
		if reportErr != nil {
			slog.Warn("the health report of the epoch root key failed", "error", reportErr)
		}
		return roots
	}
	if clearErr := r.parts.health.Clear(ctx, keyEpochRoot); clearErr != nil {
		slog.Warn("clearing the health report of the epoch root key failed", "error", clearErr)
	}
	return roots
}
