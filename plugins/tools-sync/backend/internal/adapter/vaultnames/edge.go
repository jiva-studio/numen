// Package vaultnames is the edge where the file names of the host enter Numen Sync. A note has one path in the
// plugin, its path in Unicode normal form C, whatever form the name has on the disk: a file the host lists as
// "Йога.md" in decomposed form is the note "Йога.md" in composed form, and a call that names that path reaches the
// file by the name the disk gives it. The plugin never renames a file for this; a name it makes itself is the
// composed one.
package vaultnames

import (
	"context"
	"errors"
	"slices"
	"sync"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Edge holds the names of the files of each vault that are not in normal form C. The zero value is not usable: make
// it with New.
type Edge struct {
	mu     sync.Mutex
	tables map[domain.VaultID]*table
}

// New returns an edge that knows no names yet.
func New() *Edge { return &Edge{mu: sync.Mutex{}, tables: map[domain.VaultID]*table{}} }

func (e *Edge) getTable(id domain.VaultID, feed host.FileChangeLister) *table {
	e.mu.Lock()
	defer e.mu.Unlock()
	if made, isKnown := e.tables[id]; isKnown {
		return made
	}
	made := newTable(feed)
	e.tables[id] = made
	return made
}

func isInvariant(path string) bool {
	return norm.NFC.IsNormalString(path) && norm.NFD.IsNormalString(path)
}

func getNote(path string) string { return norm.NFC.String(path) }

// table is the names of one vault: the disk name of each file whose name is not composed, found by its composed name.
type table struct {
	feed host.FileChangeLister

	mu       sync.Mutex
	cursor   host.FileChangeCursor
	isLoaded bool
	byName   map[string]int64
	bySource map[int64]string
}

func newTable(feed host.FileChangeLister) *table {
	return &table{feed: feed, mu: sync.Mutex{}, cursor: host.FileChangeCursor{FeedID: "", Seq: 0}, isLoaded: false, byName: map[string]int64{}, bySource: map[int64]string{}}
}

// getName returns the disk name of the note, which is the path itself when no other name is known.
func (t *table) getName(path string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if source, isKnown := t.byName[path]; isKnown {
		return t.bySource[source]
	}
	return path
}

func (t *table) observe(changes []host.FileChange) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.apply(changes)
}

func (t *table) apply(changes []host.FileChange) {
	for _, change := range slices.SortedStableFunc(slices.Values(changes), func(a, b host.FileChange) int { return int(a.Seq - b.Seq) }) {
		if old, isKnown := t.bySource[change.SourceID]; isKnown {
			delete(t.byName, getNote(old))
			delete(t.bySource, change.SourceID)
		}
		if change.Event == host.EventFileRemoved || getNote(change.Path) == change.Path {
			continue
		}
		t.byName[getNote(change.Path)] = change.SourceID
		t.bySource[change.SourceID] = change.Path
	}
}

// refresh brings the table up to the feed: it reads the whole vault the first time and the changes after that.
func (t *table) refresh(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.isLoaded {
		return t.load(ctx)
	}
	for {
		page, err := t.feed.ListFileChanges(ctx, host.FileChangeQuery{Since: t.cursor, Events: allEvents, Limit: 0})
		if err != nil {
			return err
		}
		if page.IsReset {
			return t.load(ctx)
		}
		t.apply(page.Changes)
		t.cursor = page.Next
		if !page.HasMore {
			return nil
		}
	}
}

var allEvents = []string{host.EventFileAdded, host.EventFileChanged, host.EventFileMoved, host.EventFileRemoved}

func (t *table) load(ctx context.Context) error {
	added := []string{host.EventFileAdded}
	head, err := t.feed.ListFileChanges(ctx, host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: "", Seq: 0}, Events: added, Limit: 0})
	if err != nil {
		return err
	}
	clear(t.byName)
	clear(t.bySource)
	query := host.FileChangeQuery{Since: host.FileChangeCursor{FeedID: head.Next.FeedID, Seq: 0}, Events: added, Limit: 0}
	for {
		page, err := t.feed.ListFileChanges(ctx, query)
		if err != nil {
			return err
		}
		if page.IsReset {
			return errors.New("the file change feed reset while the names were read")
		}
		t.apply(page.Changes)
		t.cursor, query.Since = page.Next, page.Next
		if !page.HasMore {
			t.isLoaded = true
			return nil
		}
	}
}

// run calls the host with the disk name of the path. A call that finds no file for a name that may have another form
// reads the feed for the names it has not seen and calls again when the file has another name.
func run[T any](ctx context.Context, t *table, path string, call func(name string) (T, error)) (T, error) {
	name := t.getName(path)
	out, err := call(name)
	if !errors.Is(err, host.ErrNotExist) || isInvariant(path) {
		return out, err
	}
	if t.refresh(ctx) != nil {
		return out, err
	}
	if again := t.getName(path); again != name {
		return call(again)
	}
	return out, err
}

// create calls the host with the disk name of a path that may name a file that is there already under another form.
func create[T any](ctx context.Context, t *table, path string, call func(name string) (T, error)) (T, error) {
	if !isInvariant(path) && t.getName(path) == path {
		if err := t.refresh(ctx); err != nil {
			var none T
			return none, err
		}
	}
	return call(t.getName(path))
}

var (
	_ ports.Files     = (*files)(nil)
	_ ports.Transfers = (*transfers)(nil)
	_ ports.Absent    = (*absent)(nil)
	_ ports.Notes     = (*notes)(nil)
)

func getStat(stat domain.FileStat) domain.FileStat {
	stat.Path = getNote(stat.Path)
	return stat
}
