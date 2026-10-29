package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
)

// Follow keeps the index consistent with a vault that is edited concurrently,
// and reports which notes changed.
//
// The rule is independent of the front end: changed files are reindexed, and
// when the changes cannot be determined the vault is scanned again. A window
// and a command line need the same behavior.
type Follow struct {
	watcher vaultport.VaultWatcher
	refresh Refresh
	scan    Scan

	// Changed, if set, is called each time the index and the vault are consistent
	// again.
	Changed func(VaultChanges)
	// ErrorHandler, if set, is called with each error, and with nil when a later
	// attempt succeeds, so it reflects the current state.
	ErrorHandler windowport.ErrorHandler
}

// NewFollow creates a Follow from the watcher that reports file changes, the
// refresh that updates the named files, and the scan used when the changes
// cannot be determined.
//
// Refresh and scan run from Run, which the caller starts without waiting on.
func NewFollow(watcher vaultport.VaultWatcher, refresh Refresh, scan Scan) Follow {
	return Follow{watcher: watcher, refresh: refresh, scan: scan}
}

// VaultChanges is the notification passed to a caller: the notes that changed,
// or that the whole vault must be scanned again.
type VaultChanges struct {
	Paths        []string
	ShouldReload bool
	// Assets is the paths of changed files that are not notes. Reading one is
	// separate work that takes minutes. A reload leaves Assets empty and covers
	// every asset in the vault.
	Assets []string
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
	if f.follow.Changed != nil {
		f.follow.Changed(m)
	}
}

func (f *Watch) handleError(err error) {
	if f.follow.ErrorHandler != nil {
		f.follow.ErrorHandler(err)
	}
}
