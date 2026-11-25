// Package localchanges finds the notes of a vault that changed on this device and are old enough to commit.
package localchanges

import (
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

const (
	cursorFile = "feed.json"
	settleTime = 10 * time.Second
)

// Deps are what the use case needs.
type Deps struct {
	Host  ports.Host
	State ports.Sightings
	Clock ports.Clock
}

// ScanLocalChanges follows the file change feed of a vault and lists the notes that are not what their base says.
type ScanLocalChanges struct{ deps Deps }

// NewScanLocalChanges returns the use case over the dependencies.
func NewScanLocalChanges(deps Deps) *ScanLocalChanges { return &ScanLocalChanges{deps: deps} }

// Execute returns the changed notes in the order of their paths. A note listed once is listed again until its base
// is its hash; a note written less than ten seconds ago is listed once it is older.
func (u *ScanLocalChanges) Execute(ctx context.Context, in transfer.ScanInput) ([]transfer.Change, error) {
	seen, err := u.deps.State.ReadSightings(ctx, in.VaultID)
	if err != nil {
		return nil, err
	}
	before := maps.Clone(seen)
	files := u.deps.Host.GetFiles(in.VaultID)
	if seen, err = u.follow(ctx, in.VaultID, files, seen); err != nil {
		return nil, err
	}
	found, err := u.list(ctx, files, seen, in.Bases)
	if err != nil {
		return nil, err
	}
	if maps.Equal(before, seen) {
		return found, nil
	}
	return found, u.deps.State.WriteSightings(ctx, in.VaultID, seen)
}

func (u *ScanLocalChanges) follow(ctx context.Context, id domain.VaultID, files ports.Files, seen map[string]note.FileSnapshot) (map[string]note.FileSnapshot, error) {
	events := []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}
	from := changes.Source{Feed: files, Store: u.deps.Host.Store(id), CursorFile: cursorFile}
	apply := func(ctx context.Context, batch []host.FileChange) error {
		applyChanges(seen, batch)
		return u.deps.State.WriteSightings(ctx, id, seen)
	}
	review := func(ctx context.Context) error {
		clear(seen)
		if err := readAddedFiles(ctx, files, seen); err != nil {
			return err
		}
		return u.deps.State.WriteSightings(ctx, id, seen)
	}
	return seen, changes.Follow(ctx, from, events, apply, review)
}

func readAddedFiles(ctx context.Context, files ports.Files, seen map[string]note.FileSnapshot) error {
	added := []string{host.EventFileAdded}
	head, err := files.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: "", Seq: 0}, Events: added, Limit: 0})
	if err != nil {
		return err
	}
	query := host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}, Events: added, Limit: 0}
	for {
		page, err := files.ListFileChanges(ctx, query)
		if err != nil {
			return err
		}
		if page.IsReset {
			return errFeedReset
		}
		applyChanges(seen, page.Changes)
		if !page.HasMore {
			return nil
		}
		query.Since = page.Next
	}
}

func (u *ScanLocalChanges) list(ctx context.Context, files ports.Files, seen map[string]note.FileSnapshot, bases map[string]string) ([]transfer.Change, error) {
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	var found []transfer.Change
	for _, path := range paths {
		change, err := u.scanFile(ctx, files, path, seen, bases)
		if errors.Is(err, errUnreadable) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if change != nil {
			found = append(found, *change)
		}
	}
	return found, nil
}

func (u *ScanLocalChanges) scanFile(ctx context.Context, files ports.Files, path string, seen map[string]note.FileSnapshot, bases map[string]string) (*transfer.Change, error) {
	if seen[path].Hash != "" && seen[path].Hash == bases[path] {
		return nil, nil
	}
	stat, err := files.StatFile(ctx, path)
	if errors.Is(err, host.ErrFileAbsent) {
		return nil, nil
	}
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		delete(seen, path)
		return nil, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, errUnreadable
	}
	now := u.deps.Clock.Now()
	seen[path] = updateSnapshot(seen[path], stat, now)
	if stat.Hash == bases[path] || !seen[path].IsSettled(now, settleTime) {
		return nil, nil
	}
	return &transfer.Change{Path: path, Hash: stat.Hash, Size: stat.Size, ModifiedAt: stat.ModTime.UnixNano()}, nil
}

// updateSnapshot is the snapshot of the file as the stat found it, keeping the time of the first sight of the same content.
func updateSnapshot(before note.FileSnapshot, stat domain.FileStat, now time.Time) note.FileSnapshot {
	seenAt := now.UnixNano()
	if before.Hash == stat.Hash && before.ModifiedAt == stat.ModTime.UnixNano() && before.SeenAt != 0 {
		seenAt = before.SeenAt
	}
	return note.FileSnapshot{Hash: stat.Hash, Size: stat.Size, ModifiedAt: stat.ModTime.UnixNano(), SourceID: before.SourceID, SeenAt: seenAt}
}

// errUnreadable is a file that could not be read now; the next scan reads it again.
var errUnreadable = errors.New("the file could not be read")

var errFeedReset = errors.New("the file change feed reset while the notes were read")

func applyChanges(seen map[string]note.FileSnapshot, batch []host.FileChange) {
	bySource := map[int64]string{}
	for path, one := range seen {
		bySource[one.SourceID] = path
	}
	for _, change := range batch {
		if !note.IsSyncable(change.Path) {
			continue
		}
		if change.Event == host.EventFileRemoved {
			delete(seen, change.Path)
			continue
		}
		if old, isMoved := bySource[change.SourceID]; isMoved && old != change.Path && change.Event == host.EventFileMoved {
			delete(seen, old)
		}
		bySource[change.SourceID] = change.Path
		seen[change.Path] = note.FileSnapshot{Hash: change.Hash, Size: change.Size, ModifiedAt: change.ModifiedAt, SourceID: change.SourceID, SeenAt: 0}
	}
}
