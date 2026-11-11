package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Follow keeps the index consistent with a vault that is edited concurrently,
// and reports which notes changed.
// The rule is independent of the front end: changed files are reindexed, and
// when the changes cannot be determined the vault is scanned again.
type Follow struct {
	watcher vaultport.VaultWatcher
	refresh Refreshing
	scan    Scanning

	changed      func(VaultChanges)
	errorHandler window.ErrorHandler
}

// NewFollow creates a Follow from the watcher that reports file changes, the refresh that updates the named
// files, the scan used when the changes cannot be determined, and what receives the changes and the errors;
// a nil one receives nothing. Refresh and scan run from Run, which the caller starts without waiting on.
func NewFollow(
	watcher vaultport.VaultWatcher, refresh Refreshing, scan Scanning, changed func(VaultChanges), errorHandler window.ErrorHandler,
) Follow {
	return Follow{watcher: watcher, refresh: refresh, scan: scan, changed: changed, errorHandler: errorHandler}
}

// VaultChanges is the notification passed to a caller: the notes that changed,
// or that the whole vault must be scanned again.
type VaultChanges struct {
	Paths        []string
	ShouldReload bool
	Assets       []string
}

// ShouldReadAssets reports whether any asset needs reading: an asset changed, or
// the whole vault is scanned again, which includes every asset.
func (m VaultChanges) ShouldReadAssets() bool { return m.ShouldReload || len(m.Assets) > 0 }

// Begin starts watching. Run acts on the collected changes; the two are separate
// because they occur at different times.
//
// The watch starts before the first scan, so an edit made while the vault is
// being read is buffered. A new scan waits for a walk that is already running.
func (u Follow) Begin(ctx context.Context, v domain.Vault) (*Watch, error) {
	changes, missing, err := u.watcher.WatchVault(ctx, v)
	if err != nil {
		return nil, err
	}
	return &Watch{follow: u, vault: v, changes: changes, lostChan: missing}, nil
}

// Watch is a vault being watched, whose events are not being acted on yet.
type Watch struct {
	follow   Follow
	vault    domain.Vault
	changes  <-chan []string
	lostChan <-chan struct{}
}

// Run acts on everything the watch has collected, and goes on until ctx is
// done or the watch stops.
func (f *Watch) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case paths, watching := <-f.changes:
			if !watching {
				return
			}
			res, err := f.follow.refresh.Execute(ctx, f.vault, paths)
			if err != nil {
				f.handleError(err)
				continue
			}
			f.handleError(nil)
			f.reportChanges(VaultChanges{Paths: res.GetChangedPaths(), Assets: res.Assets})

		case <-f.lostChan:
			// More changed at once than could be tracked, or the watcher lost track of the
			// changes. The vault is scanned again and listeners receive a reload notification.
			if _, err := f.follow.scan.Execute(ctx, f.vault); err != nil {
				f.handleError(err)
				continue
			}
			f.handleError(nil)
			// The scan starts from the top, so every change is among the files the walk
			// finds.
			f.reportChanges(VaultChanges{ShouldReload: true})
		}
	}
}

func (f *Watch) reportChanges(m VaultChanges) {
	if f.follow.changed != nil {
		f.follow.changed(m)
	}
}

func (f *Watch) handleError(err error) {
	if f.follow.errorHandler != nil {
		f.follow.errorHandler(err)
	}
}

// Refreshing brings the named files of a vault up to date.
type Refreshing interface {
	Execute(ctx context.Context, v domain.Vault, paths []string) (RefreshResult, error)
}

// Scanning reads a whole vault into the index.
type Scanning interface {
	Execute(ctx context.Context, v domain.Vault) (vaultreading.ScanResult, error)
}
