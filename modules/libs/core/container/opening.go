package container

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// VaultOpener is how this installation opens a vault: the walk that brings the
// index level with it, the watch that keeps it level while it is open, and the
// levelling a write asks for.
//
// Both windows open a vault through this, so a vault opened in one is a vault
// opened in the other. What each says about it while it runs is its own.
type VaultOpener struct {
	// Told, if set, is called each time the index and the vault are level again.
	Told func(VaultChanges)
	// ErrorHandler, if set, is called with what went wrong, and with nil when a
	// later attempt succeeds.
	ErrorHandler port.ErrorHandler
	// Rebuild reads every note again, whatever its fingerprint says.
	Rebuild bool

	watcher port.VaultWatcher
	scan    vault.Scan
	held    *holding
	refresh vault.Refresh
}

// VaultChanges is what an opener tells its callers: the notes that are
// different now, the files that changed and are not notes, or that the whole
// vault has to be looked at again.
//
// It is the opener's own word and not the walk's. What opens a vault here is
// the whole of what a caller is given, and a caller that had to name the
// scenario behind it to read one of these would be assembling the core itself.
type VaultChanges struct {
	Paths  []string
	Assets []string
	Reload bool
}

// Reading says whether an asset owes a read: one changed, or the whole vault is
// being looked at again and every asset with it.
func (m VaultChanges) Reading() bool { return m.Reload || len(m.Assets) > 0 }

// VaultOpener is how this installation opens a vault, the way the settings say
// one is read and watched.
func (c Config) VaultOpener(db *Index) *VaultOpener {
	return c.VaultOpenerWith(db, c.VaultReaders(), c.VaultWatcher())
}

// VaultOpenerWith is the same, with the watcher, and the readers the walk reads
// the vault through, in place of the installation's own. A note brought up to
// date after the walk is read through the installation's.
func (c Config) VaultOpenerWith(
	db *Index, walking port.VaultReaders, watcher port.VaultWatcher,
) *VaultOpener {
	scan := c.Scan(db)
	scan.Readers = walking

	held := &holding{NoteRepository: db.NotesCutAt(c.Chunking(), c.Legibility())}
	return &VaultOpener{
		watcher: watcher,
		scan:    scan,
		held:    held,
		refresh: refreshing(c, db, held),
	}
}

// refreshing brings named notes up to date, cut at this installation's sizes
// and carrying what was fetched for a link note.
func refreshing(c Config, db *Index, notes port.NoteRepository) vault.Refresh {
	refresh := vault.NewRefresh(c.VaultReaders(), db.Vaults(), notes, db.SourcesKnown(), db.Sources())
	refresh.Derived = c.DerivedStores()
	return refresh
}

// Refreshing brings named notes up to date, through whatever is following the
// vault they are in. Whatever changes a note calls it, so what changed is
// findable before the change is reported done.
func (o *VaultOpener) Refreshing() vault.Refresh { return o.refresh }

// Scanning is the walk this opener makes, for a caller asked to read the vault
// again.
func (o *VaultOpener) Scanning() vault.Scan {
	scan := o.scan
	scan.RebuildIndex = o.Rebuild
	return scan
}

// Level brings named notes up to date. A note a window writes is level before
// the answer comes back, so it is drawn as soon as it exists.
func (o *VaultOpener) Level(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.refresh.Execute(ctx, v, paths)
	return err
}

// Begin opens the vault: the watch is started, and Read is the walk beside it.
//
// A vault that cannot be watched is opened all the same, and Unwatched says why.
func (o *VaultOpener) Begin(ctx context.Context, v domain.Vault) *OpenVault {
	scan := o.Scanning()
	follow := vault.NewFollow(o.watcher, o.refresh, scan)
	if told := o.Told; told != nil {
		follow.Changed = func(m vault.VaultChanges) {
			told(VaultChanges{Paths: m.Paths, Assets: m.Assets, Reload: m.Reload})
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
	scan      vault.Scan
	follow    vault.Follow
	watching  *vault.Watch
	unwatched error
}

// Unwatched is why the vault is not being followed, and nothing while it is. A
// vault nobody is following looks exactly like a vault nothing happens to.
func (o *OpenVault) Unwatched() error { return o.unwatched }

// Read walks the vault into the index and answers how many notes it holds.
// during, if set, is called while the walk is still running.
//
// The walk writes in groups from what it read, so its copy of a note lands last
// however early the note was read. Every note brought up to date underneath it
// is read once more, and the newest copy of each lands last.
func (o *OpenVault) Read(ctx context.Context, during func()) (notes int, err error) {
	o.opening.held.begin()

	walk := o.scan
	if during != nil {
		walk.OnProgress = func(vault.ScanResult) { during() }
	}
	res, err := walk.Execute(ctx, o.vault)

	under := o.opening.held.taken()
	if err != nil {
		return res.Notes, err
	}
	if len(under) > 0 {
		if _, err := o.opening.refresh.Execute(ctx, o.vault, under); err != nil {
			o.handleError(err)
		}
	}
	return res.Notes, nil
}

// Run acts on everything the watch collects, and goes on until ctx is done or
// the watch stops. A vault whose watch could not be started runs nothing.
//
// A change named by the watch is acted on while Read is still running; reading
// the vault again waits its turn behind the walk.
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

// holding is the index, keeping the paths of the notes written through it while
// the first walk is still reading the vault.
type holding struct {
	port.NoteRepository
	writes
}

// writes is every path written through the index while a walk is reading the
// vault: each path once, in the order it was first written.
type writes struct {
	mu    sync.Mutex
	paths []string
	kept  map[string]bool
	over  bool
}

func (h *holding) Save(ctx context.Context, vaultID domain.VaultID, notes []domain.Note) error {
	for _, one := range notes {
		h.hold(one.Fingerprint.Path)
	}
	return h.NoteRepository.Save(ctx, vaultID, notes)
}

func (h *holding) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
	for _, path := range paths {
		h.hold(path)
	}
	return h.NoteRepository.Remove(ctx, vaultID, paths)
}

// begin holds the paths written through this, for the length of one walk. Every
// walk holds again: a vault read a second time is read with the same guard as
// the first.
func (w *writes) begin() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.over = false
	w.paths, w.kept = nil, nil
}

// hold takes the path before the write it belongs to, so a note whose write
// lands while the walk is still running is one of the paths taken after it.
func (w *writes) hold(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.over || w.kept[path] {
		return
	}
	if w.kept == nil {
		w.kept = map[string]bool{}
	}
	w.kept[path] = true
	w.paths = append(w.paths, path)
}

// taken is every path held, and the end of the holding: the walk is over, so a
// write that lands from now on is already the last one.
func (w *writes) taken() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.over = true
	paths := w.paths
	w.paths, w.kept = nil, nil
	return paths
}
