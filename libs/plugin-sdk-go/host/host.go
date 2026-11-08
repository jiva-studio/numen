// Package host is the plugin's view of the vault through the host: its files, its notes and links
// as the index records them, the installation's cache for it, and the plugin's own folder in the
// vault's store.
// It is also the core's shared queue of heavy work and its list of tasks.
//
// The host identifies the calling plugin by its token. A call on the vault that reads requires
// vault:read and one that changes anything requires vault:write; a call that changes the list of
// vaults or the vault the window shows requires vault:manage. The plugin's own folder requires no
// permission.
package host

import (
	"context"
	"errors"
	"io/fs"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesedit "github.com/jiva-studio/numen/libs/notes/edit"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var (
	// ErrNoVault is what a call gets while the host shows no vault.
	ErrNoVault = errors.New("no vault is shown")
	// ErrOutside is what a path that does not stay in the vault gets.
	ErrOutside = errors.New("the path is outside the vault")
	// ErrStale is what a write gets when the file changed since the baseline.
	ErrStale = notesedit.ErrStale
	// ErrAlreadyExists is what a create gets when a file is already there.
	ErrAlreadyExists = errors.New("the file already exists")
	// ErrLockedByAnother is returned when another program has locked the file.
	ErrLockedByAnother = errors.New("the file is locked by another program")
	// ErrNotANote is what a path gets when the file is not a note the vault holds.
	ErrNotANote = notesedit.ErrNotANote
	// ErrInvalidConnection is what a connection gets that is not one: a role no connection
	// carries, roles that are not inverse of each other, or a note joined to itself.
	ErrInvalidConnection = notesdomain.ErrInvalidConnection
	// ErrHierarchyCycle is what a connection gets that would put a note above itself.
	ErrHierarchyCycle = notesdomain.ErrHierarchyCycle
	// ErrNoFreeName is what a new untitled note gets when every numbered name is taken.
	ErrNoFreeName = notesdomain.ErrNoFreeName
	// ErrUnknownVault is what a call gets when it names a vault the installation does not know.
	ErrUnknownVault = errors.New("the vault is not on the list")
	// ErrNotExist is fs.ErrNotExist: a file, a note or a name that is not there.
	ErrNotExist = fs.ErrNotExist
	// ErrNotAttempted is returned for a note of a batch when the call was canceled before it was changed.
	ErrNotAttempted = errors.New("the call was canceled before this note was changed")
	// ErrNotText is returned by a ranged read for bytes that are not UTF-8.
	ErrNotText = errors.New("the file is not text")
)

type IndexStatus int

const (
	// Indexed is a vault the index carries.
	Indexed IndexStatus = iota + 1
	// Indexing is a vault being read into the index.
	Indexing
	// IndexFailed is a vault that could not be read into the index.
	IndexFailed
)

type IndexState struct {
	Status IndexStatus
	Reason string
}

// Change is what a watch reports: the files that changed, or that the host now shows another vault and
// the watch must restart. Err is set only on the last change before the channel closes, when the watch
// ended on a failure.
type Change struct {
	Paths          []string
	IsVaultChanged bool
	Err            error
}

// Move is what a watch of moves reports: that something moved in some vault. Err is set only on the
// last move before the channel closes, when the watch ended on a failure.
type Move struct {
	Err error
}

// Vault is the vault the host is showing. A plugin works it through Client; tests work it through
// hosttest.Vault.
type Vault interface {
	VaultSelector
	VaultIndex
	FileAccess
	VaultNotes
	CacheStore
}

// VaultSelector chooses which vault the other calls work on.
type VaultSelector interface {
	// GetVault names the vault, or returns ErrNoVault.
	GetVault(ctx context.Context) (domain.Vault, error)
	// GetVaults lists every vault in the installation.
	GetVaults(ctx context.Context) ([]domain.Vault, error)
	// With returns the vault with the given identity, used as the shown vault is. A call on it returns
	// ErrUnknownVault when the installation has no such vault.
	With(id domain.VaultID) Vault
}

// VaultIndex controls the index of a vault and reports changes to it.
type VaultIndex interface {
	// EnsureIndexed starts reading the vault into the index when it is not there, and returns the index
	// state. The work continues after it returns.
	EnsureIndexed(ctx context.Context) (IndexState, error)
	// WatchMoves signals that something moved in some vault, until ctx is done. It returns once the
	// watch is active, and the channel closes when the watch ends.
	WatchMoves(ctx context.Context) (<-chan Move, error)
	// Scan updates the index for the files named.
	Scan(ctx context.Context, paths []string) error
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)
	// Watch reports the files that changed until ctx is done. It returns once the watch is active, and
	// the channel closes when the watch ends.
	Watch(ctx context.Context) (<-chan Change, error)
}

// FileRange is a ranged read of a file: the text, where it begins in the file, and the file stat of
// the file it was read from.
type FileRange struct {
	Text  string
	Start int
	Stat  domain.FileStat
}

// FileReader reads the files of a vault.
type FileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	// Read returns the file with the file stat it had when it was read.
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	// ReadRange reads at most length bytes of the file from start as text, truncated at whole characters.
	// Zero or more than 16000 reads 16000. A range that is not UTF-8 returns ErrNotText; a folder returns
	// ErrNotExist.
	ReadRange(ctx context.Context, path string, start, length int) (FileRange, error)
}

// FileAccess reads and writes the files of a vault.
type FileAccess interface {
	FileReader
	// Write replaces the file if it matches the baseline, and returns ErrStale if it does not.
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	// Create makes a file that is not there, else it returns ErrAlreadyExists.
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers to this plugin until the release is called. The release returns
	// the failure to release; the host frees a lock whose release failed after its timeout.
	Lock(ctx context.Context) (release func() error, err error)
}

// VaultNotes reads the notes of a vault and the links between them, renames, moves and removes the
// notes, and edits their frontmatter links.
type VaultNotes interface {
	NoteLookup
	LinkLookup
	noteArranger
	LinkEditor
}

// NoteLookup finds notes by path, name and type.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// LinkLookup finds the links of notes.
type LinkLookup interface {
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, targets []string) (map[string]domain.ResolvedLink, error)
}

// CacheStore reads and writes the plugin's cache of a vault.
type CacheStore interface {
	// ReadCache returns what the plugin kept for this vault under the name, and false where nothing is.
	ReadCache(ctx context.Context, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, name string, content []byte) error
}
