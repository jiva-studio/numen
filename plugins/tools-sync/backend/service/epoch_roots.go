package service

import (
	"context"
	"crypto/ed25519"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/epochtrust"
)

const keyEpochRoot = "tools-sync/epoch-root"

// readEpochRoots returns the root keys that vouch for the restore epochs of the server at address: the ones of the
// build, and the key of the setting only for a server other than the default one.
func (r *runtime) readEpochRoots(address string) []ed25519.PublicKey {
	roots, err := epochtrust.GetRoots(address, r.epochRoot.Get())
	if r.parts.health == nil {
		return roots
	}
	if err != nil {
		_ = r.parts.health.Report(context.Background(), health.Item{
			Key: keyEpochRoot, Severity: health.Warning, Text: "The epoch root key of Numen Sync is not 32 bytes of hex",
			Detail: "Restore announcements of the sync server are ignored until the key is fixed.", Action: health.OpenSettings(settingEpochRoot),
		})
		return roots
	}
	_ = r.parts.health.Clear(context.Background(), keyEpochRoot)
	return roots
}
