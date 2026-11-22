package index

import (
	"context"
	"crypto/sha256"
	"slices"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
)

// SaveDelay is how long the index waits after a change before it writes its snapshot, so a burst of
// changes is one write.
const SaveDelay = 5 * time.Second

const cursorFile = "cursor"

// SnapshotVersion is the format of the kept snapshot. It changes with the shape of what the
// snapshot holds, so a snapshot of another version is not trusted and every note is read again.
const SnapshotVersion = 2

// After runs fn once after the delay and returns what cancels it. It is time.AfterFunc outside of
// tests.
type After func(delay time.Duration, fn func()) (cancel func())

// Persistence keeps the snapshot of an index between runs and follows the file change feed from the
// place the snapshot was taken at.
type Persistence struct {
	Snapshots ports.Snapshots
	Feed      ports.ChangeFeed
	After     After
	Cursors   CursorStore
}

// SystemAfter is the After of the plugin.
func SystemAfter(delay time.Duration, fn func()) func() {
	timer := time.AfterFunc(delay, fn)
	return func() { timer.Stop() }
}

type saver struct {
	Persistence
	mu     sync.Mutex
	cancel func()
	cursor host.FileChangeCursor
}

// Persist makes the index keep a snapshot of itself: written SaveDelay after a change and by Flush,
// read and brought up to date from the file change feed by Start.
func (i *Index) Persist(p Persistence) {
	i.saver = &saver{Persistence: p, mu: sync.Mutex{}, cancel: nil, cursor: host.FileChangeCursor{}}
}

// Start makes the index ready: from the snapshot of the last run and the changes made since, or by
// reading every note when there is no snapshot or the feed cannot continue from it. Without
// persistence it reads every note.
func (i *Index) Start(ctx context.Context) error {
	if i.saver == nil {
		return i.Scan(ctx)
	}
	if snapshot, isFound := i.saver.Snapshots.Load(ctx); isFound && snapshot.Version == SnapshotVersion {
		i.restore(snapshot)
	}
	if err := i.follow(ctx); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.isReady {
		i.isReady = true
		i.bump()
	}
	return nil
}

// Flush brings the index up to the end of the feed and writes its snapshot at once; a failure of the
// feed still writes what the index has, with the cursor it reached.
func (i *Index) Flush(ctx context.Context) {
	if i.saver == nil {
		return
	}
	i.saver.stop()
	if i.GetState().IsReady {
		ignore(i.follow(ctx))
	}
	i.save()
}

func (i *Index) follow(ctx context.Context) error {
	from := changes.Source{Feed: i.saver.Feed, Store: i.saver.Cursors(i.saver.getCursor, i.saver.setCursor), CursorFile: cursorFile}
	events := []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved, host.EventNoteFrontmatterChanged}
	return changes.Follow(ctx, from, events, i.applyChanges, i.Scan)
}

func (i *Index) applyChanges(ctx context.Context, batch []host.FileChange) error {
	var paths, tagPaths []string
	isChanged, isMoved := false, false
	for _, change := range batch {
		if i.isIgnored(change.Path) {
			continue
		}
		switch change.Event {
		case host.EventFileRemoved:
			isChanged = i.drop(change.Path) || isChanged
		case host.EventFileMoved:
			isMoved = true
			paths = append(paths, change.Path)
			tagPaths = append(tagPaths, change.Path)
		case host.EventFileAdded, host.EventNoteFrontmatterChanged:
			paths = append(paths, change.Path)
			tagPaths = append(tagPaths, change.Path)
		default:
			paths = append(paths, change.Path)
		}
	}
	if isMoved {
		dropped, err := i.dropMissing(ctx)
		if err != nil {
			return err
		}
		isChanged = isChanged || dropped
	}
	if err := i.checkPaths(ctx, paths, isChanged); err != nil {
		return err
	}
	return i.refreshTags(ctx, tagPaths)
}

func (i *Index) dropMissing(ctx context.Context) (bool, error) {
	listed, err := i.listPaths(ctx)
	if err != nil {
		return false, err
	}
	present := make(map[string]struct{}, len(listed))
	for _, path := range listed {
		present[path] = struct{}{}
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	isDropped := false
	for path := range i.entries {
		if _, isPresent := present[path]; !isPresent {
			i.removeEntry(path)
			isDropped = true
		}
	}
	return isDropped, nil
}

func (i *Index) restore(snapshot ports.Snapshot) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.saver.mu.Lock()
	i.saver.cursor = snapshot.Cursor
	i.saver.mu.Unlock()
	for path, saved := range snapshot.Notes {
		if isUnder(i.ignored, path) {
			continue
		}
		next := entry{stat: fromSavedStat(saved.Stat), isUnreadable: saved.IsUnreadable, tasks: saved.Tasks, tags: saved.Tags, hash: [sha256.Size]byte{}}
		copy(next.hash[:], saved.Hash)
		i.putEntry(path, next)
	}
}

func (i *Index) takeSnapshot() (ports.Snapshot, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if !i.isReady {
		return ports.Snapshot{}, false
	}
	notes := make(map[string]ports.SavedNote, len(i.entries))
	for path, one := range i.entries {
		notes[path] = ports.SavedNote{Stat: toSavedStat(one.stat), Hash: slices.Clone(one.hash[:]), Tasks: slices.Clone(one.tasks), Tags: slices.Clone(one.tags), IsUnreadable: one.isUnreadable}
	}
	i.saver.mu.Lock()
	defer i.saver.mu.Unlock()
	return ports.Snapshot{Version: SnapshotVersion, Cursor: i.saver.cursor, Notes: notes}, true
}

func (i *Index) save() {
	if snapshot, isReady := i.takeSnapshot(); isReady {
		ignore(i.saver.Snapshots.Save(snapshot))
	}
}

func (i *Index) scheduleSave() {
	if i.saver == nil {
		return
	}
	i.saver.schedule(i.save)
}

func (s *saver) schedule(save func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	s.cancel = s.After(SaveDelay, func() {
		s.mu.Lock()
		s.cancel = nil
		s.mu.Unlock()
		save()
	})
}

func (s *saver) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

// CursorStore returns the store the follower keeps its cursor in, over the cursor of the snapshot,
// so the cursor and the notes it stands for are written together.
type CursorStore func(get func() host.FileChangeCursor, set func(host.FileChangeCursor)) changes.Store

func (s *saver) getCursor() host.FileChangeCursor {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cursor
}

func (s *saver) setCursor(cursor host.FileChangeCursor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cursor = cursor
}

func ignore(_ error) {}

func toSavedStat(stat domain.FileStat) ports.SavedStat {
	return ports.SavedStat{
		Path: stat.Path, Kind: string(stat.Kind), Size: stat.Size, ModTime: stat.ModTime,
		FileID: uint64(stat.FileID), Hash: stat.Hash, Version: stat.Version,
	}
}

func fromSavedStat(stat ports.SavedStat) domain.FileStat {
	return domain.FileStat{
		Path: stat.Path, Kind: domain.SourceKind(stat.Kind), Size: stat.Size, ModTime: stat.ModTime,
		FileID: domain.FileID(stat.FileID), Hash: stat.Hash, Version: stat.Version,
	}
}
