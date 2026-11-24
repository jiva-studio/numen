// Package localchanges finds the notes of a vault that changed on this device and are old enough to commit.
package localchanges

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
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
	files := u.deps.Host.GetFiles(in.VaultID)
	if seen, err = follow(ctx, files, u.deps.Host.Store(in.VaultID), seen); err != nil {
		return nil, err
	}
	found, err := u.list(ctx, files, seen, in.Bases)
	if err != nil {
		return nil, err
	}
	return found, u.deps.State.WriteSightings(ctx, in.VaultID, seen)
}

func follow(ctx context.Context, files ports.Files, store ports.Store, seen map[string]note.Seen) (map[string]note.Seen, error) {
	events := []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}
	from := changes.Source{Feed: files, Store: store, CursorFile: cursorFile}
	apply := func(_ context.Context, batch []host.FileChange) error {
		applyChanges(seen, batch)
		return nil
	}
	review := func(ctx context.Context) error {
		clear(seen)
		return readAddedFiles(ctx, files, seen)
	}
	return seen, changes.Follow(ctx, from, events, apply, review)
}

func readAddedFiles(ctx context.Context, files ports.Files, seen map[string]note.Seen) error {
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

func (u *ScanLocalChanges) list(ctx context.Context, files ports.Files, seen map[string]note.Seen, bases map[string]string) ([]transfer.Change, error) {
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	var found []transfer.Change
	for _, path := range paths {
		if seen[path].Hash != "" && seen[path].Hash == bases[path] {
			continue
		}
		stat, err := files.StatFile(ctx, path)
		if errors.Is(err, host.ErrFileAbsent) {
			continue
		}
		if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
			delete(seen, path)
			continue
		}
		if err != nil {
			return nil, err
		}
		seen[path] = note.Seen{Hash: stat.Hash, Size: stat.Size, ModifiedAt: stat.ModTime.UnixNano(), SourceID: seen[path].SourceID}
		if stat.Hash != bases[path] && u.deps.Clock.Now().Sub(stat.ModTime) >= settleTime {
			found = append(found, transfer.Change{Path: path, Hash: stat.Hash, Size: stat.Size, ModifiedAt: stat.ModTime.UnixNano()})
		}
	}
	return found, nil
}

var errFeedReset = errors.New("the file change feed reset while the notes were read")

func applyChanges(seen map[string]note.Seen, batch []host.FileChange) {
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
		seen[change.Path] = note.Seen{Hash: change.Hash, Size: change.Size, ModifiedAt: change.ModifiedAt, SourceID: change.SourceID}
	}
}
