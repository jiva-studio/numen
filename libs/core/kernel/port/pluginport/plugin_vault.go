package pluginport

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/note/domain"
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

type pluginVaultConnectionReader interface {
	GetConnection(ctx context.Context, a, b, kind string) (domain.Connection, error)
}

type pluginVaultConnectionWriter interface {
	SaveConnection(ctx context.Context, c domain.Connection) error
	RemoveSide(ctx context.Context, from, to, kind string) error
	RemoveConnection(ctx context.Context, a, b, kind string) error
	Join(ctx context.Context, from, to string, side domain.Side) error
}

// pluginVaultRelations reads and writes the relationships between notes, with the
// rules of a connection enforced by the host.
type pluginVaultRelations interface {
	pluginVaultConnectionReader
	pluginVaultConnectionWriter
	// CreateInRelation returns the path of the note it created, which is set when
	// the note exists even if the error is not nil.
	CreateInRelation(ctx context.Context, in domain.RelatedNote) (string, error)
}

type pluginVaultFileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	// ReadRange reads at most length bytes of a file from start as text, truncated at whole
	// characters. A range that is not UTF-8 returns ErrNotText; a folder or a path not in the vault
	// returns fs.ErrNotExist.
	ReadRange(ctx context.Context, path string, start, length int) (PluginFileRange, error)
}

type PluginVaultReader interface {
	pluginVaultNoteReader
	pluginVaultLinkReader
	pluginVaultFileReader
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)
}

// PluginVaultWriter provides mutating operations on files and notes in a vault.
type PluginVaultWriter interface {
	pluginVaultArranger
	// Write replaces the file if it still matches the baseline, and returns the new file stat.
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	// Lock holds the vault's writers until the release is called.
	Lock(ctx context.Context) (release func(), err error)
	// Scan updates the index for the files named.
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

// pluginVaultArranger renames, moves and removes the notes of a vault, repairing the links that
// stop resolving, and runs the checks of the vault.
type pluginVaultArranger interface {
	// RenameNote returns the path of the note after the rename, whether the title is stored in the
	// filename, and the move of its file, nil when the file kept its path.
	RenameNote(ctx context.Context, path, title string) (newPath string, isFilenamed bool, move *domain.MoveResult, err error)
	MoveNote(ctx context.Context, from, to string) (domain.MoveResult, error)
	// RemoveNote moves the note to the vault's trash and returns its path there and the notes whose
	// links are now broken. A folder returns ErrNotANote of the notes library.
	RemoveNote(ctx context.Context, path string) (trashPath string, brokenLinks []string, err error)
	// GetProblems runs the checks named, or every check that is not opt-in when none is named, and
	// returns the names of the checks the result covers and the problems they found.
	GetProblems(ctx context.Context, checks []domain.Check) (checkNames []domain.Check, problems []domain.VaultProblem, err error)
	pluginVaultLinkEditor
}

// pluginVaultLinkEditor edits the frontmatter links of the notes of a vault.
type pluginVaultLinkEditor interface {
	// AddLinks writes links into the frontmatter links of notes, grouped by note: the links of one
	// note are one write against one file stat. The result has one entry per addition, in order.
	AddLinks(ctx context.Context, additions []PluginLinkAddition) ([]PluginLinkAdded, error)
	// UpdateLinks changes the role, type, label and note of the links the note at from has to the
	// target, keeping the target. An empty field is kept.
	UpdateLinks(ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat) (domain.FileStat, error)
	// RemoveLinks removes the frontmatter links the note at from has to the target, of every role, or
	// only those of the role when one is given.
	RemoveLinks(ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat) (domain.FileStat, error)
}

// PluginVault is the open vault as the host exposes it to a plugin: its files,
// the index data of its notes and links, the plugin's folder in its store, and
// the cache the installation keeps for a plugin.
type PluginVault interface {
	PluginVaultReader
	PluginVaultWriter
	pluginVaultRelations
	PluginVaultStore
	PluginVaultLifecycle
}

// ErrNotText is returned by a ranged read of bytes that are not UTF-8.
var ErrNotText = errors.New("the file is not text")

// PluginFileRange is a ranged read of a file: the text, its start in the file, and the file stat of
// the file it was read from.
type PluginFileRange struct {
	Text  string
	Start int
	Stat  domain.FileStat
}

// PluginLinkAddition is one link to write into the note at From, which the caller read as Baseline.
type PluginLinkAddition struct {
	From     string
	Baseline domain.FileStat
	Link     domain.Link
}

// PluginLinkAdded is the outcome of one addition: the note after the write, or the error of a link
// not written.
type PluginLinkAdded struct {
	Stat domain.FileStat
	Err  error
}
