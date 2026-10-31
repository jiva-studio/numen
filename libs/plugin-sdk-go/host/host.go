// Package host is the plugin's view of the vault through the host: its files, its notes and links
// as the index records them, the installation's cache for it, and the plugin's own folder in the
// vault's store.
// It is also the core's shared queue of heavy work and its list of tasks.
//
// The host identifies the calling plugin by its token. A call on the vault that reads requires
// vault:read and one that changes anything requires vault:write; the plugin's own folder requires no
// permission.
package host

import (
	"context"
	"errors"
	"io/fs"

	notesdomain "github.com/jiva-studio/numen/libs/notes/domain"
	notesedit "github.com/jiva-studio/numen/libs/notes/edit"
)

var (
	// ErrNoVault is what a call gets while the host shows no vault.
	ErrNoVault = errors.New("host: no vault is shown")
	// ErrOutside is what a path that does not stay in the vault gets.
	ErrOutside = errors.New("host: not a path inside the vault")
	// ErrStale is what a write gets when the file changed since the baseline.
	ErrStale = notesedit.ErrStale
	// ErrAlreadyExists is what a create gets when a file is already there.
	ErrAlreadyExists = errors.New("host: a file is already there")
	// ErrLockedByAnother is what a name gets when another program holds the file.
	ErrLockedByAnother = errors.New("host: the file is held by another program")
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
	ErrUnknownVault = errors.New("host: no such vault")
	// ErrNotExist is fs.ErrNotExist: a file, a note or a name that is not there.
	ErrNotExist = fs.ErrNotExist
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
// the watch must restart.
type Change struct {
	Paths          []string
	IsVaultChanged bool
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
	GetVault(ctx context.Context) (VaultInfo, error)
	// GetVaults lists every vault in the installation.
	GetVaults(ctx context.Context) ([]VaultInfo, error)
	// With returns the vault with the given identity, used as the shown vault is. A call on it returns
	// ErrUnknownVault when the installation has no such vault.
	With(id VaultID) Vault
}

// VaultIndex controls the index of a vault and reports changes to it.
type VaultIndex interface {
	// EnsureIndexed starts reading the vault into the index when it is not there, and returns the index
	// state. The work continues after it returns.
	EnsureIndexed(ctx context.Context) (IndexState, error)
	// WatchMoves signals that something moved in some vault, until ctx is done. It returns once the
	// watch is active, and the channel closes when the watch ends.
	WatchMoves(ctx context.Context) (<-chan struct{}, error)
	// Scan brings the index level with the files named.
	Scan(ctx context.Context, paths []string) error
	GetParseProblems(ctx context.Context) ([]VaultProblem, error)
	// Watch reports the files that changed until ctx is done. It returns once the watch is active, and
	// the channel closes when the watch ends.
	Watch(ctx context.Context) (<-chan Change, error)
}

// FileAccess reads and writes the files of a vault.
type FileAccess interface {
	StatFile(ctx context.Context, path string) (FileStat, error)
	// Read returns the file with the file stat it had when it was read.
	Read(ctx context.Context, path string) ([]byte, FileStat, error)
	// Write replaces the file if it matches the baseline, and returns ErrStale if it does not.
	Write(ctx context.Context, path string, content []byte, baseline FileStat) (FileStat, error)
	// Create makes a file that is not there, else it returns ErrAlreadyExists.
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers to this plugin until the release is called.
	Lock(ctx context.Context) (release func(), err error)
}

// VaultNotes reads the notes of a vault and the links between them.
type VaultNotes interface {
	NoteLookup
	LinkLookup
}

// NoteLookup finds notes by path, name and type.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType NoteType) ([]string, error)
}

// LinkLookup finds the links of notes.
type LinkLookup interface {
	GetLinks(ctx context.Context, from string) ([]ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, targets []string) (map[string]ResolvedLink, error)
}

// CacheStore reads and writes the plugin's cache of a vault.
type CacheStore interface {
	// ReadCache returns what the plugin kept for this vault under the name, and false where nothing is.
	ReadCache(ctx context.Context, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, name string, content []byte) error
}
