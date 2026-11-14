// Package index holds the tasks of the notes of the vault in memory. It is derived from the notes:
// dropping it and scanning again gives the same tasks.
package index

import (
	"context"

	"github.com/jiva-studio/numen/plugins/views-todos/backend/filter"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
)

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

// Index is the tasks of the notes, one entry for each note.
type Index struct{}

// New returns an empty index over the notes the reader and the lister show.
func New(_ ports.FileReader, _ ports.NoteLister, _ ParseFunc) *Index {
	return &Index{}
}

// Scan reads every note and marks the index ready when the last is read.
func (i *Index) Scan(_ context.Context) error {
	return nil
}

// Refresh checks the notes at the paths, after the renames moved their entries. No path and no
// rename is a full check.
func (i *Index) Refresh(_ context.Context, _ []string, _ []Rename) error {
	return nil
}

// SetIgnoredFolders sets the folders, separated by commas, whose notes are not searched.
func (i *Index) SetIgnoredFolders(_ context.Context, _ string) error {
	return nil
}

// List returns the tasks that the filter accepts at today.
func (i *Index) List(_ string, _ filter.Filter, _ int) Listing {
	return Listing{}
}

// Summarize counts the open tasks at today.
func (i *Index) Summarize(_ string) Summary {
	return Summary{}
}

// GetState returns the revision and the readiness now.
func (i *Index) GetState() State {
	return State{}
}

// Watch sends the state once at the start and after each change of the index, until ctx is done.
func (i *Index) Watch(_ context.Context, _ func(State) error) error {
	return nil
}
