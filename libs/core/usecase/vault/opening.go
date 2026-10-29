package vault

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// VaultOpener opens a vault for this installation: the walk that brings the
// index up to date with it, the watch that keeps it up to date while open, and
// the reconciliation a write requests.
type VaultOpener struct {
	Told          func(VaultChanges)
	ErrorHandler  windowport.ErrorHandler
	ShouldRebuild bool

	watcher  vaultport.VaultWatcher
	scan     Scan
	Buffered *BufferedNoteRepository
	Refresh  Refresh
}

// ConfigureVaultOpener constructs a VaultOpener with the given watcher, scan, buffered repository, and refresh.
func ConfigureVaultOpener(watcher vaultport.VaultWatcher, scan Scan, buffered *BufferedNoteRepository, refresh Refresh) *VaultOpener {
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

// Level brings named notes up to date.
func (o *VaultOpener) Level(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.Refresh.Execute(ctx, v, paths)
	return err
}

// Begin opens the vault: the watch is started, and Read is the walk beside it.
func (o *VaultOpener) Begin(ctx context.Context, v domain.Vault) *OpenVault {
	scan := o.GetScan()
	follow := NewFollow(o.watcher, o.Refresh, scan)
	if callback := o.Told; callback != nil {
		follow.Changed = func(m VaultChanges) {
			callback(VaultChanges{Paths: m.Paths, Assets: m.Assets, ShouldReload: m.ShouldReload})
		}
	}
	follow.ErrorHandler = o.ErrorHandler
	watching, err := follow.Begin(ctx, v)
	return &OpenVault{
		opening:   o,
		vault:     v,
		scan:      scan,
		follow:    follow,
		watching:  watching,
		unwatched: err,
	}
}

// OpenVault is one vault an application has opened.
type OpenVault struct {
	opening   *VaultOpener
	vault     domain.Vault
	scan      Scan
	follow    Follow
	watching  *Watch
	unwatched error
}

// GetUnwatchedReason is why the vault is not being followed, and nothing while it is.
func (o *OpenVault) GetUnwatchedReason() error { return o.unwatched }

// Read traverses the vault into the index and returns the number of notes it holds.
func (o *OpenVault) Read(ctx context.Context, during func()) (notes int, err error) {
	o.opening.Buffered.StartRecording()

	walk := o.scan
	if during != nil {
		walk.OnProgress = func(ScanResult) { during() }
	}
	res, err := walk.Execute(ctx, o.vault)

	recSession := o.opening.Buffered.StopRecording()
	if err != nil {
		return res.Notes, err
	}
	if _, err := o.opening.Refresh.Execute(ctx, o.vault, recSession); err != nil {
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
	if o.follow.ErrorHandler != nil {
		o.follow.ErrorHandler(err)
	}
}

// BufferedNoteRepository is the index, keeping the paths of the notes written through it while
// the first walk is still reading the vault.
type BufferedNoteRepository struct {
	noteport.NoteRepository
	BufferedWrites
}

// BufferedWrites is every path written through the index while a walk is reading the vault.
type BufferedWrites struct {
	mu       sync.Mutex
	paths    []string
	recorded map[string]bool
	isOver   bool
}

func (b *BufferedNoteRepository) Save(ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks) error {
	for _, one := range notes {
		b.Record(one.Note.FileStat.Path)
	}
	return b.NoteRepository.Save(ctx, vaultID, notes)
}

func (b *BufferedNoteRepository) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
	for _, path := range paths {
		b.Record(path)
	}
	return b.NoteRepository.Remove(ctx, vaultID, paths)
}

// StartRecording holds the paths written through this, for the length of one walk.
func (w *BufferedWrites) StartRecording() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isOver = false
	w.paths, w.recorded = nil, nil
}

// Record takes the path before the write it belongs to.
func (w *BufferedWrites) Record(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isOver || w.recorded[path] {
		return
	}
	if w.recorded == nil {
		w.recorded = map[string]bool{}
	}
	w.recorded[path] = true
	w.paths = append(w.paths, path)
}

// StopRecording is every path held, and the end of the holding.
func (w *BufferedWrites) StopRecording() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isOver = true
	paths := w.paths
	w.paths, w.recorded = nil, nil
	return paths
}
