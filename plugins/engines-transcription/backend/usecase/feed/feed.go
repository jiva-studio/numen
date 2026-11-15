// Package feed starts the transcription of the recordings of a vault that the file change feed
// names as added or changed. It keeps the recordings it started in the plugin's folder of the
// vault's store, by the number the index keeps each file under, until their transcriptions end, so
// a transcription the plugin stopped in starts again when the plugin runs next.
package feed

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"

	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/recording"
)

// CursorFile is the file of the plugin's store folder the cursor of the feed of a vault is kept in.
const CursorFile = "feed.cursor.json"

// PendingFile is the file of the plugin's store folder the recordings whose transcription has not
// ended are kept in.
const PendingFile = "feed.pending.json"

const pendingVersion = 1

// State is what a start did for a file.
type State int

const (
	// StateStarted is a transcription the start began.
	StateStarted State = iota + 1
	// StateRunning is a transcription that was running already.
	StateRunning
	// StateDone is a file whose transcription is the live text version for its bytes now.
	StateDone
)

// Runs starts the transcriptions of the files of a vault.
type Runs interface {
	// Start starts the transcription of the file unless one runs or its text is made.
	Start(ctx context.Context, vault domain.VaultID, path string) (State, error)
	IsRunning(ctx context.Context, vault domain.VaultID, path string) (bool, error)
}

// Vault is the feed of one vault and the plugin's folder of its store.
type Vault struct {
	Feed  host.FileChangeLister
	Store changes.Store
}

var events = []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}

// Follower brings the transcriptions of each vault up to date with its file change feed.
type Follower struct {
	open   func(id domain.VaultID) Vault
	runs   Runs
	report func(what string, err error)
	mu     sync.Mutex
	vaults map[domain.VaultID]*vaultRecordings
}

// vaultRecordings is what the follower knows of one vault. pending is the recordings whose
// transcription has not ended, by the number the index keeps each under; started is the paths this
// process started or found running. isLoaded is false until PendingFile was read, and again after an
// update failed.
type vaultRecordings struct {
	mu       sync.Mutex
	isLoaded bool
	pending  map[int64]string
	started  map[string]bool
}

// New returns the follower of the vaults open opens, starting the transcriptions through runs. report
// receives the failure of a start, which is tried again on the next update.
func New(open func(id domain.VaultID) Vault, runs Runs, report func(what string, err error)) *Follower {
	return &Follower{open: open, runs: runs, report: report, mu: sync.Mutex{}, vaults: map[domain.VaultID]*vaultRecordings{}}
}

func (f *Follower) getVault(id domain.VaultID) *vaultRecordings {
	f.mu.Lock()
	defer f.mu.Unlock()
	one, ok := f.vaults[id]
	if !ok {
		one = &vaultRecordings{mu: sync.Mutex{}, isLoaded: false, pending: map[int64]string{}, started: map[string]bool{}}
		f.vaults[id] = one
	}
	return one
}

// Execute reads the feed of the vault from the cursor, keeps the recordings added or changed since,
// and starts the transcription of each kept recording that has none running and no text for its
// bytes now. A reset of the feed walks every recording of the vault.
func (f *Follower) Execute(ctx context.Context, id domain.VaultID) error {
	one := f.getVault(id)
	one.mu.Lock()
	defer one.mu.Unlock()
	u := update{follower: f, id: id, vault: f.open(id), recordings: one}
	from := changes.Source{Feed: u.vault.Feed, Store: u.vault.Store, CursorFile: CursorFile}
	if err := one.load(ctx, u.vault.Store, from); err != nil {
		return err
	}
	if err := changes.Follow(ctx, from, events, u.apply, u.review); err != nil {
		one.isLoaded = false
		return fmt.Errorf("follow the file changes: %w", err)
	}
	return u.startPending(ctx)
}

func (v *vaultRecordings) load(ctx context.Context, store changes.Store, from changes.Source) error {
	if v.isLoaded {
		return nil
	}
	pending, err := readPending(ctx, store)
	if err != nil {
		return err
	}
	if pending == nil {
		if err := changes.Forget(ctx, from); err != nil {
			return err
		}
		pending = map[int64]string{}
	}
	v.pending, v.isLoaded = pending, true
	return nil
}

// update is one Execute over one vault.
type update struct {
	follower   *Follower
	id         domain.VaultID
	vault      Vault
	recordings *vaultRecordings
}

func (u update) apply(ctx context.Context, batch []host.FileChange) error {
	u.keep(batch)
	return writePending(ctx, u.vault.Store, u.recordings.pending)
}

func (u update) keep(batch []host.FileChange) {
	for _, change := range batch {
		switch change.Event {
		case host.EventFileAdded, host.EventFileChanged:
			delete(u.recordings.started, change.Path)
			u.keepRecording(change)
		case host.EventFileMoved:
			u.keepRecording(change)
		case host.EventFileRemoved:
			delete(u.recordings.pending, change.SourceID)
		}
	}
}

func (u update) keepRecording(change host.FileChange) {
	if recording.IsRecording(change.Path) {
		u.recordings.pending[change.SourceID] = change.Path
		return
	}
	delete(u.recordings.pending, change.SourceID)
}

// review keeps every recording of the vault: the feed holds the latest addition and change of every
// file it has, from the start of the feed.
func (u update) review(ctx context.Context) error {
	head, err := u.vault.Feed.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{}, Events: events})
	if err != nil {
		return fmt.Errorf("read the file changes: %w", err)
	}
	clear(u.recordings.pending)
	since := host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}
	for {
		page, err := u.vault.Feed.ListFileChanges(ctx, host.FileChangeQuery{
			Since: since, Events: []string{host.EventFileAdded, host.EventFileChanged},
		})
		if err != nil {
			return fmt.Errorf("read the files: %w", err)
		}
		if page.IsReset {
			u.follower.report("walk the recordings", errors.New("the file change feed no longer holds every file"))
			break
		}
		u.keep(page.Changes)
		since = page.Next
		if !page.HasMore {
			break
		}
	}
	return writePending(ctx, u.vault.Store, u.recordings.pending)
}

func (u update) startPending(ctx context.Context) error {
	ids := slices.SortedFunc(maps.Keys(u.recordings.pending), cmp.Compare[int64])
	isChanged := false
	for _, id := range ids {
		isEnded, err := u.startRecording(ctx, u.recordings.pending[id])
		if err != nil {
			return err
		}
		if isEnded {
			delete(u.recordings.pending, id)
			isChanged = true
		}
	}
	if !isChanged {
		return nil
	}
	return writePending(ctx, u.vault.Store, u.recordings.pending)
}

// startRecording starts the transcription of a kept recording, and returns true when the recording is
// no longer kept: its transcription, started by this process, ended, or its text is made.
func (u update) startRecording(ctx context.Context, path string) (bool, error) {
	if u.recordings.started[path] {
		isRunning, err := u.follower.runs.IsRunning(ctx, u.id, path)
		if err != nil {
			return false, err
		}
		if !isRunning {
			delete(u.recordings.started, path)
		}
		return !isRunning, nil
	}
	state, err := u.follower.runs.Start(ctx, u.id, path)
	if err != nil {
		u.follower.report("transcribe "+path, err)
		return false, nil
	}
	if state == StateDone {
		return true, nil
	}
	u.recordings.started[path] = true
	return false, nil
}

// pendingFile is the content of PendingFile. A missing file, or one of another version, holds no
// recordings, and the next update walks every recording of the vault.
type pendingFile struct {
	Version    int              `json:"version"`
	Recordings map[int64]string `json:"recordings"`
}

func readPending(ctx context.Context, store changes.Store) (map[int64]string, error) {
	content, err := store.Read(ctx, PendingFile)
	if errors.Is(err, host.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", PendingFile, err)
	}
	var stored pendingFile
	if json.Unmarshal(content, &stored) != nil || stored.Version != pendingVersion {
		return nil, nil
	}
	if stored.Recordings == nil {
		stored.Recordings = map[int64]string{}
	}
	return stored.Recordings, nil
}

func writePending(ctx context.Context, store changes.Store, pending map[int64]string) error {
	content, err := json.Marshal(pendingFile{Version: pendingVersion, Recordings: pending})
	if err != nil {
		return err
	}
	if err := store.Write(ctx, PendingFile, content); err != nil {
		return fmt.Errorf("write %s: %w", PendingFile, err)
	}
	return nil
}
