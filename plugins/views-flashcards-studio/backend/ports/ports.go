// Package ports is what the use cases of the plugin ask of the vault they work, the answers they keep
// between launches, and the clock.
//
// A use case orchestrates these and knows nothing of how the host answers them.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/review"
)

var (
	// ErrStale is what a write says when the file it was about to replace is no longer the one the
	// caller read.
	ErrStale = edit.ErrStale
	// ErrNotANote is what a path gets when there is a file at it and the vault does not hold it as a
	// note.
	ErrNotANote = edit.ErrNotANote
	// ErrAlreadyExists is what making a file says when one is already there.
	ErrAlreadyExists = errors.New("a file is already there")
	// ErrHeldByAnother is what a name gets when a program outside this one has the file.
	ErrHeldByAnother = errors.New("the file is held by another program")
)

// Clock is what time it is. A use case that takes none has none.
type Clock func() time.Time

// VaultReader is what a use case reads of one vault's files.
type VaultReader interface {
	// StatFile says what the file is without opening it. It returns fs.ErrNotExist for a file
	// that is not there and ErrNotANote for one the vault does not hold as a note.
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the bytes of the file.
	Read(ctx context.Context, path string) ([]byte, error)
}

// VaultReaders opens one vault's files for reading. Which vault a use case works on is decided while
// it runs.
type VaultReaders interface {
	Open(v domain.Vault) (VaultReader, error)
}

// VaultWriter is what a use case changes in one vault's files.
type VaultWriter interface {
	// Write replaces the file if it is still what the file stat says, else it returns ErrStale. An
	// empty file stat holds the write to nothing.
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	// Create makes a file that is not there, else it returns ErrAlreadyExists.
	Create(ctx context.Context, path string, content []byte) error
}

// VaultWriters opens one vault's files for writing.
type VaultWriters interface {
	Open(v domain.Vault) (VaultWriter, error)
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context, v domain.Vault) (release func(), err error)
}

// NoteQueries is what the index says of a vault's notes.
type NoteQueries interface {
	// GetNotes names the notes at the paths, and leaves out a path with no note.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
	// GetTypes says what type each note at the paths is.
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
	// GetPathsOfType lists the notes of a type, by path.
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
	// GetCardModels lists the card models of the vault, by path.
	GetCardModels(ctx context.Context, vaultID domain.VaultID) ([]CardModel, error)
	// GetNamedPaths lists the notes filed under a name.
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
	// Has says whether the index carries the vault.
	Has(ctx context.Context, vaultID domain.VaultID) (bool, error)
}

// LinkQueries is what the index says of a vault's links.
type LinkQueries interface {
	GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)
	// ResolveLinks says where each address written in a note lands, keyed by what was written. An
	// address that reaches nothing is absent.
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, rawLinks []string) (map[string]domain.ResolvedLink, error)
}

// ProblemQueries is what parsing a vault's files turned up.
type ProblemQueries interface {
	GetParseProblems(ctx context.Context, vaultID domain.VaultID) ([]domain.VaultProblem, error)
}

// Entry is one file of review answers: the run that wrote it, and how many bytes it holds.
type Entry struct {
	Run  string
	Size int
}

// LogStore is one vault's files of review answers, one to a run, which the plugin cannot make
// again.
type LogStore interface {
	// ReadLog returns the answers a run wrote.
	ReadLog(ctx context.Context, run string) ([]byte, error)
	// AppendLog adds to the answers of a run, making its file if it is not there.
	AppendLog(ctx context.Context, run string, content []byte) error
	// ListLogs reports the files of review answers.
	ListLogs(ctx context.Context) ([]Entry, error)
}

// LogStores opens one vault's files of review answers.
type LogStores interface {
	Open(v domain.Vault) (LogStore, error)
}

// ScheduleStore keeps what was worked out for a vault between launches. It belongs to the
// installation and can be made again.
type ScheduleStore interface {
	// Read returns what is kept, or fs.ErrNotExist.
	Read(ctx context.Context, vaultID domain.VaultID) ([]byte, error)
	Write(ctx context.Context, vaultID domain.VaultID, content []byte) error
}

// ScheduleAlgorithm is which algorithm schedules a card, at the share of the cards a preset asks for.
type ScheduleAlgorithm interface {
	GetName() string
	CreateScheduler(retention float64) review.Scheduler
}

// CardModel is one card model as a caller choosing between them sees it: where the file is, and what it is
// called.
type CardModel struct {
	Path  string
	Title string
}
