package container

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// Opening is how this installation opens a vault: the walk that brings the
// index level with it, the watch that keeps it level while it is open, and the
// levelling a write asks for.
//
// Both windows open a vault through this, so a vault opened in one is a vault
// opened in the other. What each says about it while it runs is its own.
type Opening struct {
	// Told, if set, is called each time the index and the vault are level again.
	Told func(vault.Moved)
	// Trouble, if set, is called with what went wrong, and with nil when a later
	// attempt succeeds.
	Trouble func(error)
	// Rebuild reads every note again, whatever its fingerprint says.
	Rebuild bool

	watcher port.VaultWatcher
	scan    vault.Scan
	held    *holding
	refresh vault.Refresh
}

// Opening is how this installation opens a vault, the way the settings say one
// is read and watched.
func (c Config) Opening(db *Index) *Opening {
	return c.OpeningWith(db, c.VaultReaders(), c.VaultWatcher())
}

// OpeningWith is the same, with the watcher, and the readers the walk reads the
// vault through, in place of the installation's own. A note brought up to date
// after the walk is read through the installation's.
func (c Config) OpeningWith(
	db *Index, walking port.VaultReaders, watcher port.VaultWatcher,
) *Opening {
	scan := c.Scan(db)
	scan.Readers = walking

	held := &holding{NoteRepository: db.NotesCutAt(c.Cutting())}
	return &Opening{
		watcher: watcher,
		scan:    scan,
		held:    held,
		refresh: vault.Refresh{Readers: c.VaultReaders(), Notes: held},
	}
}

// Refreshing brings named notes up to date, through whatever is following the
// vault they are in. Whatever changes a note calls it, so what changed is
// findable before the change is reported done.
func (o *Opening) Refreshing() vault.Refresh { return o.refresh }

// Scanning is the walk this opening makes, for a caller asked to read the vault
// again.
func (o *Opening) Scanning() vault.Scan {
	scan := o.scan
	scan.RebuildIndex = o.Rebuild
	return scan
}

// Level brings named notes up to date. A note a window writes is level before
// the answer comes back, so it is drawn as soon as it exists.
func (o *Opening) Level(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := o.refresh.Execute(ctx, v, paths)
	return err
}

// Begin opens the vault: the watch is started, and Read is the walk beside it.
//
// A vault that cannot be watched is opened all the same, and Unwatched says why.
func (o *Opening) Begin(ctx context.Context, v domain.Vault) *Open {
	scan := o.Scanning()
	follow := vault.Follow{
		Watcher: o.watcher,
		Refresh: o.refresh,
		Scan:    scan,
		Changed: o.Told,
		Trouble: o.Trouble,
	}
	watching, err := follow.Begin(ctx, v)
	return &Open{
		opening:   o,
		vault:     v,
		scan:      scan,
		follow:    follow,
		watching:  watching,
		unwatched: err,
	}
}

// Open is one vault an application has opened.
type Open struct {
	opening   *Opening
	vault     domain.Vault
	scan      vault.Scan
	follow    vault.Follow
	watching  *vault.Following
	unwatched error
}

// Unwatched is why the vault is not being followed, and nothing while it is. A
// vault nobody is following looks exactly like a vault nothing happens to.
func (o *Open) Unwatched() error { return o.unwatched }

// Read walks the vault into the index, handing back how far it has got as it
// goes.
//
// The walk writes in groups from what it read, so its copy of a note lands last
// however early the note was read. Every note brought up to date underneath it
// is read once more, and the newest copy of each lands last.
func (o *Open) Read(ctx context.Context, got func(vault.ScanResult)) (vault.ScanResult, error) {
	o.opening.held.begin()

	walk := o.scan
	walk.OnProgress = got
	res, err := walk.Execute(ctx, o.vault)

	under := o.opening.held.taken()
	if err != nil {
		return res, err
	}
	if len(under) > 0 {
		if _, err := o.opening.refresh.Execute(ctx, o.vault, under); err != nil {
			o.trouble(err)
		}
	}
	return res, nil
}

// Run acts on everything the watch collects, and goes on until ctx is done or
// the watch stops. A vault whose watch could not be started runs nothing.
func (o *Open) Run(ctx context.Context) {
	if o.watching == nil {
		return
	}
	o.watching.Run(ctx)
}

func (o *Open) trouble(err error) {
	if o.follow.Trouble != nil {
		o.follow.Trouble(err)
	}
}

// holding is the index, keeping the paths of the notes written through it while
// the first walk is still reading the vault.
type holding struct {
	port.NoteRepository

	mu    sync.Mutex
	paths []string
	kept  map[string]bool
	over  bool
}

func (h *holding) Save(ctx context.Context, vaultID string, notes []domain.Note) error {
	for _, n := range notes {
		h.hold(n.Ref.Path)
	}
	return h.NoteRepository.Save(ctx, vaultID, notes)
}

func (h *holding) Remove(ctx context.Context, vaultID string, paths []string) error {
	for _, path := range paths {
		h.hold(path)
	}
	return h.NoteRepository.Remove(ctx, vaultID, paths)
}

// begin holds the paths written through this, for the length of one walk. Every
// walk holds again: a vault read a second time is read with the same guard as
// the first.
func (h *holding) begin() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.over = false
	h.paths, h.kept = nil, nil
}

// hold takes the path before the write it belongs to, so a note whose write
// lands while the walk is still running is one of the paths taken after it.
func (h *holding) hold(path string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.over || h.kept[path] {
		return
	}
	if h.kept == nil {
		h.kept = map[string]bool{}
	}
	h.kept[path] = true
	h.paths = append(h.paths, path)
}

// taken is every path held, and the end of the holding: the walk is over, so a
// write that lands from now on is already the last one.
func (h *holding) taken() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.over = true
	paths := h.paths
	h.paths, h.kept = nil, nil
	return paths
}
