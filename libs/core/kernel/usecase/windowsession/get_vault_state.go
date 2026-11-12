package windowsession

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ProgressCounter counts how many chunks of a vault can carry a vector under a recipe and how many carry one.
type ProgressCounter interface {
	HasProgress() bool
	CountProgress(ctx context.Context, vaultID domain.VaultID, recipe string) (total, embedded int64, err error)
}

// GetVaultState reads what the window knows about the vault it shows: whether the first scan is done,
// what stopped it, and how far the vault has been read for meaning.
func GetVaultState(ctx context.Context, rt *Runtime, progress ProgressCounter) (window.VaultState, error) {
	showing := rt.GetCurrentVault()
	name, recipe := rt.GetModel()
	out := window.VaultState{
		Vault:       showing,
		IsReady:     rt.IsReady(),
		Error:       rt.GetError(),
		WatchError:  rt.GetWatchError(),
		IsEmbedding: name != "",
	}
	if progress != nil && progress.HasProgress() && showing.ID != "" {
		target, embedded, err := progress.CountProgress(ctx, showing.ID, recipe)
		if err != nil {
			return out, err
		}
		out.ChunkCount, out.EmbeddedCount = target, embedded
	}
	return out, nil
}
