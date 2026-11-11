package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vaultreading"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// WriteRecorder holds the paths written through the index while a walk reads a vault.
type WriteRecorder interface {
	// StartRecording starts holding the paths, for the length of one walk.
	StartRecording()
	// StopRecording returns every path held and ends the holding.
	StopRecording() []string
}

// VaultOpener opens a vault for this installation: the walk that brings the
// index up to date with it, the watch that keeps it up to date while open, and
// the reconciliation a write requests.
type VaultOpener struct {
	ShouldRebuild bool

	watcher  vaultport.VaultWatcher
	scan     Scan
	Buffered WriteRecorder
	Refresh  Refresh
}

// ConfigureVaultOpener constructs a VaultOpener with the given watcher, scan, buffered repository, and refresh.
func ConfigureVaultOpener(watcher vaultport.VaultWatcher, scan Scan, buffered WriteRecorder, refresh Refresh) *VaultOpener {
	return &VaultOpener{
		watcher:  watcher,
		scan:     scan,
		Buffered: buffered,
		Refresh:  refresh,
	}
}

// GetRefresh brings named notes up to date, through whatever is following the vault they are in.
func (o *VaultOpener) GetRefresh() Refresh { return o.Refresh }

// GetScan returns the walk this opener makes, for a caller that re-reads the vault.
func (o *VaultOpener) GetScan() Scan {
	scan := o.scan
	scan.ShouldRebuildIndex = o.ShouldRebuild
	return scan
}

// UpdateIndex updates the index for the named notes.
func (o *VaultOpener) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.Refresh.Execute(ctx, v, paths)
	return err
}

// Begin opens the vault: the watch is started, and Read is the walk beside it. changed receives what the
// watch collects and errorHandler the errors it meets; a nil one receives nothing.
func (o *VaultOpener) Begin(
	ctx context.Context, v domain.Vault, changed func(VaultChanges), errorHandler window.ErrorHandler,
) *OpenVault {
	scan := o.GetScan()
	follow := NewFollow(o.watcher, o.Refresh, scan, changed, errorHandler)
	open := &OpenVault{
		buffered:     o.Buffered,
		refresh:      o.Refresh,
		scan:         scan,
		vault:        v,
		errorHandler: errorHandler,
	}
	watching, err := follow.Begin(ctx, v)
	if err == nil {
		open.watching = watching
	}
	open.unwatched = err
	return open
}

// OpenVault is one vault an application has opened.
type OpenVault struct {
	buffered     WriteRecorder
	refresh      refreshing
	scan         progressScanning
	vault        domain.Vault
	errorHandler window.ErrorHandler
	watching     runner
	unwatched    error
}

// progressScanning reads a whole vault into the index and reports its progress.
type progressScanning interface {
	ScanWithProgress(
		ctx context.Context, v domain.Vault, onProgress func(vaultreading.ScanResult),
	) (vaultreading.ScanResult, error)
}

// runner acts on what a watch collects.
type runner interface {
	Run(ctx context.Context)
}

// GetUnwatchedReason is why the vault is not being followed, and nothing while it is.
func (o *OpenVault) GetUnwatchedReason() error { return o.unwatched }

// Read traverses the vault into the index and returns the number of notes it holds.
func (o *OpenVault) Read(ctx context.Context, during func()) (notes int, err error) {
	o.buffered.StartRecording()

	var onProgress func(vaultreading.ScanResult)
	if during != nil {
		onProgress = func(vaultreading.ScanResult) { during() }
	}
	res, err := o.scan.ScanWithProgress(ctx, o.vault, onProgress)

	recSession := o.buffered.StopRecording()
	if err != nil {
		return res.Notes, err
	}
	if _, err := o.refresh.Execute(ctx, o.vault, recSession); err != nil {
		o.handleError(err)
	}
	return res.Notes, nil
}

// Run acts on everything the watch collects, and goes on until ctx is done or the watch stops.
func (o *OpenVault) Run(ctx context.Context) {
	if o.watching == nil {
		return
	}
	o.watching.Run(ctx)
}

func (o *OpenVault) handleError(err error) {
	if o.errorHandler != nil {
		o.errorHandler(err)
	}
}

// UpdateIndex updates the index for the named notes.
func (o *OpenVault) UpdateIndex(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.refresh.Execute(ctx, v, paths)
	return err
}
