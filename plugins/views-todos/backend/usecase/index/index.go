// Package index holds the tasks of the notes of the vault in memory. It is derived from the notes:
// dropping it and scanning again gives the same tasks.
package index

import (
	"context"
	"crypto/sha256"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

// maxReads is how many notes are read at once.
const maxReads = 8

// ParseFunc reads the tasks of a note. todo.ParseNote is the one the plugin uses.
type ParseFunc func(raw []byte) ([]todo.Task, error)

// Rename is a note that moved.
type Rename struct {
	From, To string
}

// Summary counts the open tasks.
type Summary struct {
	IsReady         bool
	OverdueCount    int
	TodayCount      int
	NoDateCount     int
	UnreadableCount int
}

// Listing is the tasks a filter accepts, limited to a count. Total is the count before the limit.
type Listing struct {
	Tasks    []todo.Task
	Total    int
	Revision uint64
}

// State is the revision and readiness that a watcher receives.
type State struct {
	Revision uint64
	IsReady  bool
}

// entry is what the index knows of one note: the file stat and the hash of the bytes it parsed, and
// the tasks. A note that cannot be read or parsed has no tasks and is flagged.
type entry struct {
	stat         domain.FileStat
	hash         [sha256.Size]byte
	tasks        []todo.Task
	isUnreadable bool
}

// Index is the tasks of the notes, one entry for each note.
type Index struct {
	files ports.FileReader
	notes ports.NoteLister
	parse ParseFunc

	mu       sync.Mutex
	entries  map[string]entry
	ignored  []string
	revision uint64
	isReady  bool
	changed  chan struct{}
}

// New returns an empty index over the notes the reader and the lister show.
func New(files ports.FileReader, notes ports.NoteLister, parse ParseFunc) *Index {
	return &Index{
		files: files, notes: notes, parse: parse,
		mu: sync.Mutex{}, entries: map[string]entry{}, ignored: nil, revision: 0, isReady: false, changed: make(chan struct{}),
	}
}

// NewForNotes returns an empty index that reads its notes with todo.ParseNote.
func NewForNotes(files ports.FileReader, notes ports.NoteLister) *Index {
	return New(files, notes, todo.ParseNote)
}

// Scan reads every note and marks the index ready when the last is read.
func (i *Index) Scan(ctx context.Context) error {
	paths, err := i.listPaths(ctx)
	if err != nil {
		return err
	}
	if _, err := i.visit(ctx, paths, i.readNote); err != nil {
		return err
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	for path := range i.entries {
		if !slices.Contains(paths, path) {
			delete(i.entries, path)
		}
	}
	i.isReady = true
	i.bump()
	return nil
}

// Refresh checks the notes at the paths, after the renames moved their entries. No path and no
// rename is a full check.
func (i *Index) Refresh(ctx context.Context, paths []string, renames []Rename) error {
	if len(paths) == 0 && len(renames) == 0 {
		if !i.GetState().IsReady {
			return i.Scan(ctx)
		}
		return i.checkAll(ctx)
	}
	i.mu.Lock()
	moved := i.moveEntries(renames)
	i.mu.Unlock()
	checked := make([]string, 0, len(paths))
	for _, path := range paths {
		if !i.isIgnored(path) {
			checked = append(checked, path)
		}
	}
	return i.checkPaths(ctx, checked, len(moved) > 0)
}

// SetIgnoredFolders sets the folders, separated by commas, whose notes are not searched.
func (i *Index) SetIgnoredFolders(ctx context.Context, folders string) error {
	var prefixes []string
	for folder := range strings.SplitSeq(folders, ",") {
		folder = strings.TrimSuffix(strings.TrimSpace(folder), "/")
		if folder != "" {
			prefixes = append(prefixes, folder)
		}
	}
	i.mu.Lock()
	i.ignored = prefixes
	isReady := i.isReady
	if i.dropIgnored() {
		i.bump()
	}
	i.mu.Unlock()
	if !isReady {
		return nil
	}
	return i.checkAll(ctx)
}

// List returns the tasks that the filter accepts at today.
func (i *Index) List(today string, f filter.Filter, limit int) Listing {
	i.mu.Lock()
	var all []todo.Task
	for _, one := range i.entries {
		all = append(all, one.tasks...)
	}
	revision := i.revision
	i.mu.Unlock()
	accepted := filter.Apply(all, f, today)
	total := len(accepted)
	if limit > 0 && total > limit {
		accepted = accepted[:limit]
	}
	return Listing{Tasks: accepted, Total: total, Revision: revision}
}

// Summarize counts the open tasks at today.
func (i *Index) Summarize(today string) Summary {
	i.mu.Lock()
	defer i.mu.Unlock()
	summary := Summary{IsReady: i.isReady, OverdueCount: 0, TodayCount: 0, NoDateCount: 0, UnreadableCount: 0}
	for _, one := range i.entries {
		if one.isUnreadable {
			summary.UnreadableCount++
		}
		for _, task := range one.tasks {
			switch {
			case task.Status != todo.StatusOpen:
			case task.Due == "":
				summary.NoDateCount++
			case task.Due < today:
				summary.OverdueCount++
			case task.Due == today:
				summary.TodayCount++
			}
		}
	}
	return summary
}

// GetState returns the revision and the readiness now.
func (i *Index) GetState() State {
	i.mu.Lock()
	defer i.mu.Unlock()
	return State{Revision: i.revision, IsReady: i.isReady}
}

// Watch sends the state once at the start and after each change of the index, until ctx is done.
// A reader that is slow gets the latest state, not each one in between.
func (i *Index) Watch(ctx context.Context, send func(State) error) error {
	for {
		i.mu.Lock()
		state, changed := State{Revision: i.revision, IsReady: i.isReady}, i.changed
		i.mu.Unlock()
		if err := send(state); err != nil {
			return err
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return nil
		}
	}
}

func (i *Index) bump() {
	i.revision++
	close(i.changed)
	i.changed = make(chan struct{})
}

func (i *Index) listPaths(ctx context.Context) ([]string, error) {
	listed, err := i.notes.GetPathsOfType(ctx, domain.TypeNote)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, path := range listed {
		if !i.isIgnored(path) {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func (i *Index) isIgnored(path string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return isUnder(i.ignored, path)
}

func isUnder(folders []string, path string) bool {
	return slices.ContainsFunc(folders, func(folder string) bool {
		return path == folder || strings.HasPrefix(path, folder+"/")
	})
}

func (i *Index) dropIgnored() bool {
	isDropped := false
	for path := range i.entries {
		if isUnder(i.ignored, path) {
			delete(i.entries, path)
			isDropped = true
		}
	}
	return isDropped
}

func (i *Index) moveEntries(renames []Rename) []string {
	var moved []string
	for _, rename := range renames {
		one, found := i.entries[rename.From]
		if !found {
			continue
		}
		delete(i.entries, rename.From)
		for n := range one.tasks {
			one.tasks[n].Path = rename.To
		}
		i.entries[rename.To] = one
		moved = append(moved, rename.To)
	}
	return moved
}

func (i *Index) checkAll(ctx context.Context) error {
	paths, err := i.listPaths(ctx)
	if err != nil {
		return err
	}
	i.mu.Lock()
	isChanged := false
	for path := range i.entries {
		if !slices.Contains(paths, path) {
			delete(i.entries, path)
			isChanged = true
		}
	}
	i.mu.Unlock()
	return i.checkPaths(ctx, paths, isChanged)
}
