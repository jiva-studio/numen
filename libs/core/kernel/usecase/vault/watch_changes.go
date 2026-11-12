package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// WatchChanges follows a vault and reports the paths that change in it.
type WatchChanges struct {
	watcher vaultport.VaultWatcher
}

func NewWatchChanges(watcher vaultport.VaultWatcher) *WatchChanges {
	return &WatchChanges{watcher: watcher}
}

// Execute starts watching the vault until ctx is done. changes carries the paths of each change, and
// lost is closed once the watch cannot follow the vault any longer.
func (u *WatchChanges) Execute(ctx context.Context, v domain.Vault) (changes <-chan []string, lost <-chan struct{}, err error) {
	return u.watcher.WatchVault(ctx, v)
}
