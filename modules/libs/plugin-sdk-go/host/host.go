// Package host is the vault as a plugin works it through the host: its files, its notes and links
// as the index knows them, the folder of its own data, and the cache the installation keeps for it.
//
// The host answers each call as the plugin it knows by its token, and holds a call that reads to
// vault:read and a call that changes anything to vault:write.
package host

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/edit"
)

var (
	// ErrNoVault is what a call gets while the host shows no vault.
	ErrNoVault = errors.New("host: no vault is shown")
	// ErrOutside is what a path that does not stay in the vault gets.
	ErrOutside = errors.New("host: not a path inside the vault")
	// ErrStale is what a write gets when the file changed since the baseline.
	ErrStale = edit.ErrStale
	// ErrOccupied is what a create gets when a file is already there.
	ErrOccupied = errors.New("host: a file is already there")
	// ErrHeldByAnother is what a name gets when another program holds the file.
	ErrHeldByAnother = errors.New("host: the file is held by another program")
	// ErrNoDataFolder is what a data call gets when the manifest names no data folder.
	ErrNoDataFolder = errors.New("host: the manifest names no data folder")
	// ErrNotANote is what a path gets when the file is not a note the vault holds.
	ErrNotANote = edit.ErrNotANote
	// ErrUnknownVault is what a call gets when it names a vault the installation does not know.
	ErrUnknownVault = errors.New("host: no such vault")
	// ErrNotExist is fs.ErrNotExist: a file, a note or a name that is not there.
	ErrNotExist = fs.ErrNotExist
)

// DataFile is a file in the folder the plugin keeps its data in.
type DataFile struct {
	Name string
	Size int64
}

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

// Change is what a watch tells: the files that changed, or that the host now shows another vault and
// the watch has to begin again.
type Change struct {
	Paths          []string
	IsVaultChanged bool
}

// Vault is the vault the host is showing. A plugin works it through Client; tests work it through
// hosttest.Vault.
type Vault interface {
	// GetVault names the vault, or returns ErrNoVault.
	GetVault(ctx context.Context) (domain.Vault, error)
	// GetVaults lists every vault the installation knows.
	GetVaults(ctx context.Context) ([]domain.Vault, error)
	// With is the vault of the identity, worked as the shown vault is. A call on it returns
	// ErrUnknownVault when the installation knows no such vault.
	With(id domain.VaultID) Vault
	// EnsureIndexed starts reading the vault into the index when it is not there, and says where
	// that stands. The work goes on after it returns.
	EnsureIndexed(ctx context.Context) (IndexState, error)
	// SetInUse tells the window that the plugin is working the vault for the person: a session has
	// begun on it.
	SetInUse(ctx context.Context) error
	// WatchMoves tells that something moved in some vault, until ctx is done. It returns once the
	// watch is on, and the channel closes when the watch ends.
	WatchMoves(ctx context.Context) (<-chan struct{}, error)

	GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error)
	// Read returns the file with the fingerprint it had when it was read.
	Read(ctx context.Context, path string) ([]byte, domain.Fingerprint, error)
	// Write replaces the file if it is still what the baseline says, else it returns ErrStale.
	Write(ctx context.Context, path string, content []byte, baseline domain.Fingerprint) (domain.Fingerprint, error)
	// Create makes a file that is not there, else it returns ErrOccupied.
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers to this plugin until the release is called.
	Lock(ctx context.Context) (release func(), err error)
	// Scan brings the index level with the files named.
	Scan(ctx context.Context, paths []string) error

	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, targets []string) (map[string]domain.ResolvedLink, error)
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)

	// ReadData reads a file of the plugin's folder in the vault's own store.
	ReadData(ctx context.Context, name string) ([]byte, error)
	AppendData(ctx context.Context, name string, content []byte) error
	ListData(ctx context.Context) ([]DataFile, error)

	// ReadCache returns what the plugin kept for this vault under the name, and false where nothing is.
	ReadCache(ctx context.Context, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, name string, content []byte) error

	// Watch tells of the files that changed until ctx is done. It returns once the watch is on, and
	// the channel closes when the watch ends.
	Watch(ctx context.Context) (<-chan Change, error)
}
