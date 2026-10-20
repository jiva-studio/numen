package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// PluginDataFile is a file in the folder a plugin keeps its own data in.
type PluginDataFile struct {
	Name string
	Size int
}

// PluginDataStore is the folder one plugin keeps what it cannot make again in, inside a vault's
// own store. It answers for that folder and no other.
type PluginDataStore interface {
	// Read returns what is kept under the name, or fs.ErrNotExist.
	Read(ctx context.Context, name string) ([]byte, error)
	// Append adds to the file of that name, making it if it is not there.
	Append(ctx context.Context, name string, content []byte) error
	// List returns the files directly in the folder, sorted by name.
	List(ctx context.Context) ([]PluginDataFile, error)
}

// PluginDataStores opens the folder of a plugin in the store of a vault.
type PluginDataStores interface {
	Open(v domain.Vault, folder string) (PluginDataStore, error)
}

// PluginCache keeps what a plugin worked out for a vault between launches. It belongs to the
// installation and not to the vault, and can be made again.
type PluginCache interface {
	// Read returns what is kept under the name, and false where nothing is.
	Read(ctx context.Context, pluginID string, vaultID domain.VaultID, name string) ([]byte, bool, error)
	// Write replaces what is kept under the name, whole.
	Write(ctx context.Context, pluginID string, vaultID domain.VaultID, name string, content []byte) error
}

// PluginIndexStatus is where a vault stands in the index.
type PluginIndexStatus int

const (
	// PluginIndexed is a vault the index carries.
	PluginIndexed PluginIndexStatus = iota + 1
	// PluginIndexing is a vault being read into the index.
	PluginIndexing
	// PluginIndexFailed is a vault that could not be read into the index.
	PluginIndexFailed
)

// PluginIndexState is where a vault stands in the index, and why it failed when it did.
type PluginIndexState struct {
	Status PluginIndexStatus
	Reason string
}

// PluginVault is the vault the window is showing, as the host works it for a plugin: its files,
// what the index says of its notes and links, the folder of its store a plugin keeps data in, and
// the cache the installation keeps for a plugin.
type PluginVault interface {
	// GetVault is the vault the window is showing, and false while it shows none.
	GetVault() (domain.Vault, bool)
	// GetVaults lists every vault the installation knows.
	GetVaults() ([]domain.Vault, error)
	// With is the vault of the identity, worked the way the shown vault is. It returns
	// fs.ErrNotExist for an identity the installation does not know.
	With(id domain.VaultID) (PluginVault, error)
	// EnsureIndexed brings the vault into the index when it is not there, and says where that
	// stands. The work goes on after it returns.
	EnsureIndexed(ctx context.Context) (PluginIndexState, error)
	// WatchMoves tells that something moved in some vault, until ctx is done.
	WatchMoves(ctx context.Context) (<-chan struct{}, error)

	GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error)
	// Read returns the file and what it was when it was asked about, before its bytes were read.
	Read(ctx context.Context, path string) ([]byte, domain.Fingerprint, error)
	// Write replaces the file if it is still what the baseline says, and returns what it is now.
	Write(ctx context.Context, path string, content []byte, baseline domain.Fingerprint) (domain.Fingerprint, error)
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context) (release func(), err error)
	// Scan brings the index level with the files named.
	Scan(ctx context.Context, paths []string) error

	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, written []string) (map[string]domain.ResolvedLink, error)
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)

	// OpenData opens the folder of the store of the vault.
	OpenData(folder string) (PluginDataStore, error)
	ReadCache(ctx context.Context, pluginID, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, pluginID, name string, content []byte) error

	// Watch tells of the files of the vault that changed, until ctx is done or the watch is lost.
	Watch(ctx context.Context) (changes <-chan []string, lost <-chan struct{}, err error)
}
