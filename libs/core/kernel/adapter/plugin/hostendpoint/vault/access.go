package vault

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// NoteLookup reads the index data of the notes of the vault.
type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// LinkLookup reads the links between the notes of the vault.
type LinkLookup interface {
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, from string, output []string) (map[string]domain.ResolvedLink, error)
}

type FileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	// ReadRange reads at most length bytes of a file from start as text, truncated at whole
	// characters. A range that is not UTF-8 returns plugin.ErrNotText; a folder or a path not in the vault
	// returns fs.ErrNotExist.
	ReadRange(ctx context.Context, path string, start, length int) (plugin.FileRange, error)
}

type FileWriter interface {
	// WriteFile replaces the file if it still matches the baseline, and returns the new file stat.
	WriteFile(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	CreateFile(ctx context.Context, path string, content []byte) error
	// ScanFiles updates the index for the files named.
	ScanFiles(ctx context.Context, paths []string) error
}

// Arranger renames, moves and removes the notes of the vault, repairing the links that stop
// resolving, and runs the checks of the vault.
type Arranger interface {
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
	GetParseProblems(ctx context.Context) ([]domain.VaultProblem, error)
}

// LinkEditor edits the frontmatter links of the notes of the vault.
type LinkEditor interface {
	// AddLinks writes links into the frontmatter links of notes, grouped by note: the links of one
	// note are one write against one file stat. The result has one entry per addition, in order.
	AddLinks(ctx context.Context, additions []plugin.LinkAddition) ([]plugin.LinkAdded, error)
	// UpdateLinks changes the role, type, label and note of the links the note at from has to the
	// target, keeping the target. An empty field is kept.
	UpdateLinks(ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat) (domain.FileStat, error)
	// RemoveLinks removes the frontmatter links the note at from has to the target, of every role, or
	// only those of the role when one is given.
	RemoveLinks(ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat) (domain.FileStat, error)
}

// Watch tells of the moves in the vaults of the installation.
type Watch interface {
	WatchMoves(ctx context.Context) (<-chan struct{}, error)
}

// Switch names the vault a call works and the vaults the installation lists.
type Switch interface {
	GetVault() (domain.Vault, bool)
	GetVaults() ([]domain.Vault, error)
	// With returns the capabilities over the vault with the identity.
	With(id domain.VaultID) (*Access, error)
	EnsureIndexed(ctx context.Context) (plugin.IndexState, error)
}

// StoreReader reads one plugin's own folder in the store of a vault, `.numen/<plugin id>/`. A name is a
// file directly in that folder; one that holds a slash or begins with a dot gets vault.ErrOutside.
type StoreReader interface {
	Read(ctx context.Context, name string) ([]byte, error)
	List(ctx context.Context) ([]plugin.StoreFile, error)
	Open(ctx context.Context, name string) (io.ReadSeekCloser, int64, error)
}

// StoreWriter changes one plugin's own folder in the store of a vault, under the names of StoreReader.
type StoreWriter interface {
	Append(ctx context.Context, name string, content []byte) error
	Write(ctx context.Context, name string, content []byte) error
	Delete(ctx context.Context, name string) error
	Rename(ctx context.Context, from, to string) error
}

// StoreOpener opens the folder of a plugin in the store of the vault.
type StoreOpener interface {
	OpenStore(pluginID string) (StoreReader, error)
	OpenStoreWriter(pluginID string) (StoreWriter, error)
}

// CacheStore keeps what a plugin worked out for the vault between launches.
type CacheStore interface {
	ReadCache(ctx context.Context, pluginID, name string) ([]byte, bool, error)
	WriteCache(ctx context.Context, pluginID, name string, content []byte) error
}

// Access is the open vault as the host exposes it to a plugin: its files, the index data of its
// notes and links, the plugin's folder in its store, and the cache the installation keeps for a plugin.
// Each handler uses the capabilities it needs.
type Access struct {
	Notes      NoteLookup
	Links      LinkLookup
	Files      FileReader
	Writer     FileWriter
	Arranger   Arranger
	LinkEditor LinkEditor
	Watch      Watch
	Switch     Switch
	Stores     StoreOpener
	Cache      CacheStore
	Locks      Locks
}

// Locks holds the writers of a vault for a plugin, until the plugin releases them or the hold ends.
type Locks interface {
	// Execute waits while another holder has the writers of the vault, holds them, and returns the id of the hold.
	Execute(ctx context.Context, v domain.Vault) (string, error)
	// Release gives the writers of the hold back.
	Release(id string)
}

// List is the installation's list of vaults and the vault the window shows, changed by the
// plugins that have vault:manage. The window provides it; a host with no window has none.
type List interface {
	// AddVault adds the folder at the absolute root. An empty name means the folder's name.
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	// RemoveVault removes the vault from the list and from the index. The folder stays.
	RemoveVault(ctx context.Context, id domain.VaultID) error
	// OpenVault shows the vault in the window and returns once the window shows it. A rejected open
	// returns plugin.ErrUnsavedChanges, ErrUnavailable, or an unknown-vault error.
	OpenVault(ctx context.Context, id domain.VaultID) error
	// IsMissing reports whether the vault's folder is missing.
	IsMissing(v domain.Vault) bool
}
