package hosttest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

var _ host.FileChangeLister = (*Vault)(nil)

// fileChangePageSize is the most changes one read of the feed returns, as the host has it.
const fileChangePageSize = 1000

// heldFile is what the feed knows of a file: the number the index keeps it under, its content and
// its frontmatter.
type heldFile struct {
	id          int64
	hash        [sha256.Size]byte
	frontmatter string
}

type changeKey struct {
	id    int64
	event string
}

// feed is the file change feed of a vault, as the host keeps it: the latest event of each kind of
// each file, numbered in the order the files changed.
type feed struct {
	id      string
	resets  int
	lastSeq int64
	trimmed int64
	lastID  int64
	files   map[string]heldFile
	changes map[changeKey]int64
	paths   map[int64]string
}

func newFeed() feed {
	return feed{id: "feed-0", files: map[string]heldFile{}, changes: map[changeKey]int64{}, paths: map[int64]string{}}
}

func (f *feed) record(id int64, event string) {
	f.lastSeq++
	f.changes[changeKey{id: id, event: event}] = f.lastSeq
}

func (f *feed) recordRemoved(id int64) {
	for key := range f.changes {
		if key.id == id {
			delete(f.changes, key)
		}
	}
	f.record(id, host.EventFileRemoved)
}

func (f *feed) observe(filePath string, content []byte, isPresent bool, frontmatter map[string]any) {
	held, isHeld := f.files[filePath]
	if !isPresent {
		if isHeld {
			delete(f.files, filePath)
			f.recordRemoved(held.id)
		}
		return
	}
	now := heldFile{id: held.id, hash: sha256.Sum256(content), frontmatter: encodeFrontmatter(frontmatter)}
	switch {
	case !isHeld:
		f.lastID++
		now.id = f.lastID
		f.paths[now.id] = filePath
		f.record(now.id, host.EventFileAdded)
		if now.frontmatter != "" {
			f.record(now.id, host.EventNoteFrontmatterChanged)
		}
	default:
		if now.hash != held.hash {
			f.record(now.id, host.EventFileChanged)
		}
		if now.frontmatter != held.frontmatter {
			f.record(now.id, host.EventNoteFrontmatterChanged)
		}
	}
	f.files[filePath] = now
}

func (f *feed) move(from, target string) {
	held, ok := f.files[from]
	if !ok {
		return
	}
	if displaced, isTaken := f.files[target]; isTaken && target != from {
		f.recordRemoved(displaced.id)
	}
	delete(f.files, from)
	f.files[target] = held
	f.paths[held.id] = target
	f.record(held.id, host.EventFileMoved)
}

func (f *feed) getCursor() host.FileChangeCursor {
	return host.FileChangeCursor{FeedID: f.id, Seq: f.lastSeq}
}

func (f *feed) getSourceID(filePath string) int64 { return f.files[filePath].id }

func encodeFrontmatter(frontmatter map[string]any) string {
	if frontmatter == nil {
		return ""
	}
	raw, err := json.Marshal(frontmatter)
	if err != nil {
		return fmt.Sprint(frontmatter)
	}
	return string(raw)
}

// ResetFeed starts the feed anew, as a rebuilt index does: every cursor read before resets.
func (v *Vault) ResetFeed() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	v.feed.resets++
	v.feed.id = fmt.Sprintf("feed-%d", v.feed.resets)
}

// TrimRemovals drops every removal from the feed, as the host does with removals six months old: a
// cursor before the last of them resets.
func (v *Vault) TrimRemovals() {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	for key, seq := range v.feed.changes {
		if key.event == host.EventFileRemoved {
			v.feed.trimmed = max(v.feed.trimmed, seq)
			delete(v.feed.changes, key)
		}
	}
}

// ListFileChanges returns a page of the feed after the cursor, as the host does.
func (v *Vault) ListFileChanges(ctx context.Context, q host.FileChangeQuery) (host.FileChangePage, error) {
	v.world.mu.Lock()
	defer v.world.mu.Unlock()
	if err := v.checkContext(ctx); err != nil {
		return host.FileChangePage{}, err
	}
	if q.Limit < 0 || q.Limit > fileChangePageSize || len(q.Events) == 0 || slices.ContainsFunc(q.Events, isUnknownEvent) {
		return host.FileChangePage{}, host.ErrInvalidFileChangeQuery
	}
	limit := q.Limit
	if limit == 0 {
		limit = fileChangePageSize
	}
	head := v.feed.getCursor()
	if q.Since.FeedID != head.FeedID || (q.Since.Seq > 0 && q.Since.Seq < v.feed.trimmed) {
		return host.FileChangePage{Next: head, IsReset: true}, nil
	}
	changes := v.feed.listChanges(q.Since.Seq, q.Events)
	for i := range changes {
		v.fillContent(&changes[i])
	}
	if len(changes) <= limit {
		return host.FileChangePage{Changes: changes, Next: head}, nil
	}
	changes = changes[:limit]
	return host.FileChangePage{Changes: changes, Next: host.FileChangeCursor{FeedID: head.FeedID, Seq: changes[limit-1].Seq}, HasMore: true}, nil
}

func (v *Vault) fillContent(change *host.FileChange) {
	if change.Event == host.EventFileRemoved {
		return
	}
	at, err := v.files.stat(change.Path)
	if err != nil {
		return
	}
	at, err = v.hashed(at)
	if err != nil {
		return
	}
	change.Hash, change.Size = at.Hash, at.Size
	if !at.ModTime.IsZero() {
		change.ModifiedAt = at.ModTime.UnixNano()
	}
}

func (f *feed) listChanges(seq int64, events []string) []host.FileChange {
	var changes []host.FileChange
	for key, at := range f.changes {
		if at > seq && slices.Contains(events, key.event) {
			changes = append(changes, host.FileChange{SourceID: key.id, Path: f.paths[key.id], Event: key.event, Seq: at})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Seq < changes[j].Seq })
	return changes
}

func isUnknownEvent(event string) bool {
	return !slices.Contains([]string{
		host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved, host.EventNoteFrontmatterChanged,
	}, event)
}
