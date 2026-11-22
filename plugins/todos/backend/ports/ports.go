// Package ports holds the small interfaces the use cases need from the host.
package ports

import (
	"context"

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
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
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

// SavedNote is what the index knows of one note, as it is kept between runs.
type SavedNote struct {
	Stat         domain.FileStat `json:"stat"`
	Hash         []byte          `json:"hash"`
	Tasks        []todo.Task     `json:"tasks"`
	IsUnreadable bool            `json:"isUnreadable"`
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
