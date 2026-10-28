package pluginport

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// PluginStoreFile is a file of a plugin's folder: its name, and how many bytes it holds.
type PluginStoreFile struct {
	Name string
	Size int64
}

// PluginStore is one plugin's own folder in a vault's store, `.numen/<plugin id>/`. A name is a file
// directly in that folder; one that holds a slash or begins with a dot gets vaultport.ErrOutside.
type pluginStoreMutator interface {
	Append(ctx context.Context, name string, content []byte) error
	Write(ctx context.Context, name string, content []byte) error
	Delete(ctx context.Context, name string) error
	Rename(ctx context.Context, from, to string) error
}

type PluginStore interface {
	pluginStoreMutator
	Read(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context) ([]PluginStoreFile, error)
	Open(ctx context.Context, name string) (io.ReadSeekCloser, int64, error)
}

// PluginStores opens a plugin's folder in the store of a vault.
type PluginStores interface {
	Open(v domain.Vault, pluginID string) (PluginStore, error)
	// DeleteFolder deletes the folder of the plugin and every file in it. A
	// folder that is not there is deleted.
	DeleteFolder(ctx context.Context, v domain.Vault, pluginID string) error
}

// PluginCache keeps what a plugin worked out for a vault between launches. It belongs to the
// installation and not to the vault, and can be made again.
type PluginCache interface {
	// Read returns what is kept under the name, and false where nothing is.
	Read(ctx context.Context, pluginID string, vaultID domain.VaultID, name string) ([]byte, bool, error)
	// Write replaces what is kept under the name, whole.
	Write(ctx context.Context, pluginID string, vaultID domain.VaultID, name string, content []byte) error
}

type PluginIndexStatus int

const (
	// PluginIndexed is a vault the index carries.
	PluginIndexed PluginIndexStatus = iota + 1
	// PluginIndexing is a vault being read into the index.
	PluginIndexing
	// PluginIndexFailed is a vault that could not be read into the index.
	PluginIndexFailed
)

type PluginIndexState struct {
	Status PluginIndexStatus
	Reason string
}

// PluginVaultReader provides read access to files, notes, links, and index problems of a vault.
type pluginVaultNoteReader interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

type pluginVaultLinkReader interface {
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, output []string) (map[string]domain.ResolvedLink, error)
}

type PluginVaultReader interface {
	pluginVaultNoteReader
	pluginVaultLinkReader
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)
}

// PluginVaultWriter provides mutating operations on files in a vault.
type PluginVaultWriter interface {
	// Write replaces the file if it is still what the baseline says, and returns what it is now.
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context) (release func(), err error)
	// Scan brings the index level with the files named.
	Scan(ctx context.Context, paths []string) error
}

// PluginVaultStore provides the plugin folders of the vault's store and the cache.
type PluginVaultStore interface {
	// OpenStore opens the folder of the plugin in the store of the vault.
	OpenStore(pluginID string) (PluginStore, error)
	ReadCache(ctx context.Context, pluginID, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, pluginID, name string, content []byte) error
}

// PluginVaultLifecycle manages vault identity, indexing state, usage, and file watching.
type pluginVaultWatchers interface {
	WatchMoves(ctx context.Context) (<-chan struct{}, error)
	Watch(ctx context.Context) (changes <-chan []string, lost <-chan struct{}, err error)
}

type PluginVaultLifecycle interface {
	pluginVaultWatchers
	GetVault() (domain.Vault, bool)
	GetVaults() ([]domain.Vault, error)
	With(id domain.VaultID) (PluginVault, error)
	EnsureIndexed(ctx context.Context) (PluginIndexState, error)
}

// PluginVault is the open vault as the host exposes it to a plugin: its files,
// what the index says of its notes and links, the plugin's folder in its store, and
// the cache the installation keeps for a plugin.
type PluginVault interface {
	PluginVaultReader
	PluginVaultWriter
	PluginVaultStore
	PluginVaultLifecycle
}
