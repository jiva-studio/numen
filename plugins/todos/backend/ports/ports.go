// Package ports holds the small interfaces the use cases need from the host.
package ports

import (
	"context"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

// FileReader reads the notes of the vault.
type FileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the file with the file stat it had when it was read.
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
}

// NoteLister is the one call of host.NoteLookup that the index makes.
type NoteLister interface {
	ListPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// Frontmatter reads the frontmatter of the notes of the vault.
type Frontmatter interface {
	ReadFrontmatter(ctx context.Context, q host.FrontmatterQuery) (host.FrontmatterPage, error)
}

// FileWriter writes a note over the file stat it had when it was read.
type FileWriter interface {
	// Write replaces the file if it matches the baseline, and returns host.ErrStale if it does not.
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
}

// NoteStore keeps the tasks of a note the plugin wrote, with the file stat after the write.
type NoteStore interface {
	StoreNote(path string, content []byte, stat domain.FileStat)
}

// ChangeFeed reads the file changes of the vault after a cursor.
type ChangeFeed = host.FileChangeLister

// SavedStat is the file stat of a note as the snapshot keeps it.
type SavedStat struct {
	Path    string    `json:"path"`
	Kind    string    `json:"kind"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	FileID  uint64    `json:"fileId"`
	Hash    string    `json:"hash"`
	Version uint64    `json:"version"`
}

// SavedNote is what the index knows of one note, as it is kept between runs.
type SavedNote struct {
	Stat         SavedStat   `json:"stat"`
	Hash         []byte      `json:"hash"`
	Tasks        []todo.Task `json:"tasks"`
	Tags         []string    `json:"tags"`
	IsUnreadable bool        `json:"isUnreadable"`
}

// Snapshot is the index of a run and the place of the file change feed it was brought up to.
type Snapshot struct {
	Cursor host.FileChangeCursor `json:"cursor"`
	Notes  map[string]SavedNote  `json:"notes"`
}

// Snapshots keeps the snapshot of the index between runs. It holds nothing that cannot be
// recomputed: a snapshot that cannot be read is no snapshot.
type Snapshots interface {
	Load(ctx context.Context) (Snapshot, bool)
	Save(snapshot Snapshot) error
}

// Clock tells the time the query functions and the tools compare due dates with.
type Clock interface {
	Now() time.Time
}

// Log records the failures the plugin carries on after.
type Log interface {
	Error(message string, err error)
}

// WriteLock makes the writes of the notes run one after the other, whichever use case makes them.
type WriteLock struct {
	mu sync.Mutex
}

// NewWriteLock returns an unlocked lock.
func NewWriteLock() *WriteLock {
	return &WriteLock{mu: sync.Mutex{}}
}

// Lock waits until no other write holds the lock.
func (l *WriteLock) Lock() { l.mu.Lock() }

// Unlock lets the next write run.
func (l *WriteLock) Unlock() { l.mu.Unlock() }
