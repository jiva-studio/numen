// Package feed starts the recognition of the scans of a vault that the file change feed names as
// added or changed. It keeps the scans it started in the plugin's folder of the vault's store, by the
// number the index keeps each file under, until their recognitions end, so a recognition the plugin
// stopped in starts again when the plugin runs next.
package feed

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/pending"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/scan"
)

// CursorFile is the file of the plugin's store folder the cursor of the feed of a vault is kept in.
const CursorFile = "feed.cursor.json"

// PendingFile is the file of the plugin's store folder the scans whose recognition has not ended are
// kept in.
const PendingFile = "feed.pending.json"

// State is what a start did for a file.
type State int

const (
	// StateStarted is a recognition the start began.
	StateStarted State = iota + 1
	// StateRunning is a recognition that was running already.
	StateRunning
	// StateDone is a file whose recognition is the live text version for its bytes now.
	StateDone
)

// Runs starts the recognitions of the files of a vault.
type Runs interface {
	// Start starts the recognition of the file unless one runs or its text is made.
	Start(ctx context.Context, vault domain.VaultID, path string) (State, error)
	IsRunning(ctx context.Context, vault domain.VaultID, path string) (bool, error)
}

// Vault is the feed of one vault and the plugin's folder of its store.
type Vault struct {
	Feed  host.FileChangeLister
	Store changes.Store
}

func getEvents() []string {
	return []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}
}

// Follower brings the recognitions of each vault up to date with its file change feed.
type Follower struct {
	open   func(id domain.VaultID) Vault
	runs   Runs
	report func(what string, err error)
	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultScans
}

// vaultScans is what the follower knows of one vault. pending is the scans whose recognition has not
// ended, by the number the index keeps each under; started is the paths this process started or found
// running. isLoaded is false until PendingFile was read, and again after an update failed.
type vaultScans struct {
	mu       sync.Mutex
	isLoaded bool
	pending  map[int64]string
	started  map[string]bool
}

// New returns the follower of the vaults open opens, starting the recognitions through runs. report
// receives the failure of a start, which is tried again on the next update.
func New(open func(id domain.VaultID) Vault, runs Runs, report func(what string, err error)) *Follower {
	return &Follower{open: open, runs: runs, report: report, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultScans{}}
}

func (f *Follower) getVault(id domain.VaultID) *vaultScans {
	f.mu.Lock()
	defer f.mu.Unlock()
	one, ok := f.vaults[id]
	if !ok {
		one = &vaultScans{mu: sync.Mutex{}, isLoaded: false, pending: map[int64]string{}, started: map[string]bool{}}
		f.vaults[id] = one
	}
	return one
}

// Execute reads the feed of the vault from the cursor, keeps the scans added or changed since, and
// starts the recognition of each kept scan that has none running and no text for its bytes now. A
// reset of the feed walks every scan of the vault.
func (f *Follower) Execute(ctx context.Context, id domain.VaultID) error {
	one := f.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	u := update{follower: f, id: id, vault: f.open(id), scans: one}
	from := changes.Source{Feed: u.vault.Feed, Store: u.vault.Store, CursorFile: CursorFile}
	if err := one.load(ctx, u.vault.Store, from); err != nil {
		return err
	}
	if err := changes.Follow(ctx, from, getEvents(), u.apply, u.review); err != nil {
		one.isLoaded = false
		return fmt.Errorf("follow the file changes: %w", err)
	}
	return u.startPendingRuns(ctx)
}

func (v *vaultScans) load(ctx context.Context, store changes.Store, from changes.Source) error {
	if v.isLoaded {
		return nil
	}
	files, ok, err := readPendingFiles(ctx, store)
	if err != nil {
		return err
	}
	if !ok {
		if err := changes.Forget(ctx, from); err != nil {
			return err
		}
		files = map[int64]string{}
	}
	v.pending, v.isLoaded = files, true
	return nil
}

// update is one Execute over one vault.
type update struct {
	follower *Follower
	id       domain.VaultID
	vault    Vault
	scans    *vaultScans
}

func (u update) apply(ctx context.Context, batch []host.FileChange) error {
	u.keep(batch)
	return writePendingFiles(ctx, u.vault.Store, u.scans.pending)
}

func (u update) keep(batch []host.FileChange) {
	for _, change := range batch {
		switch change.Event {
		case host.EventFileAdded, host.EventFileChanged:
			delete(u.scans.started, change.Path)
			u.keepScan(change)
		case host.EventFileMoved:
			u.keepScan(change)
		case host.EventFileRemoved:
			delete(u.scans.pending, change.SourceID)
		}
	}
}

func (u update) keepScan(change host.FileChange) {
	if scan.IsScan(change.Path) {
		u.scans.pending[change.SourceID] = change.Path
		return
	}
	delete(u.scans.pending, change.SourceID)
}

func (u update) review(ctx context.Context) error {
	head, err := u.vault.Feed.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{}, Events: getEvents()})
	if err != nil {
		return fmt.Errorf("read the file changes: %w", err)
	}
	clear(u.scans.pending)
	since := host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}
	for {
		page, err := u.vault.Feed.ListFileChanges(ctx, host.FileChangeQuery{
			Since: since, Events: []string{host.EventFileAdded, host.EventFileChanged},
		})
		if err != nil {
			return fmt.Errorf("read the files: %w", err)
		}
		if page.IsReset {
			u.follower.report("walk the scans", errors.New("the file change feed no longer holds every file"))
			break
		}
		u.keep(page.Changes)
		since = page.Next
		if !page.HasMore {
			break
		}
	}
	return writePendingFiles(ctx, u.vault.Store, u.scans.pending)
}

func (u update) startPendingRuns(ctx context.Context) error {
	ids := slices.SortedFunc(maps.Keys(u.scans.pending), cmp.Compare[int64])
	isChanged := false
	for _, id := range ids {
		isEnded, err := u.startScan(ctx, u.scans.pending[id])
		if err != nil {
			return err
		}
		if isEnded {
			delete(u.scans.pending, id)
			isChanged = true
		}
	}
	if !isChanged {
		return nil
	}
	return writePendingFiles(ctx, u.vault.Store, u.scans.pending)
}

func (u update) startScan(ctx context.Context, path string) (bool, error) {
	if u.scans.started[path] {
		isRunning, err := u.follower.runs.IsRunning(ctx, u.id, path)
		if err != nil {
			return false, err
		}
		if !isRunning {
			delete(u.scans.started, path)
		}
		return !isRunning, nil
	}
	state, err := u.follower.runs.Start(ctx, u.id, path)
	if err != nil {
		u.follower.report("recognize "+path, err)
		return false, nil
	}
	if state == StateDone {
		return true, nil
	}
	u.scans.started[path] = true
	return false, nil
}

func readPendingFiles(ctx context.Context, store changes.Store) (map[int64]string, bool, error) {
	content, err := store.Read(ctx, PendingFile)
	if errors.Is(err, host.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", PendingFile, err)
	}
	files, ok := pending.Parse(content)
	return files, ok, nil
}

func writePendingFiles(ctx context.Context, store changes.Store, files map[int64]string) error {
	content, err := pending.Encode(files)
	if err != nil {
		return err
	}
	if err := store.Write(ctx, PendingFile, content); err != nil {
		return fmt.Errorf("write %s: %w", PendingFile, err)
	}
	return nil
}
