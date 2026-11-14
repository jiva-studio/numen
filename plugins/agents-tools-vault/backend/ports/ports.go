// Package ports declares what the use cases of the vault tools require. The host clients of the SDK
// satisfy each interface but NotePathResolver.
package ports

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// FileReader reads the files of one vault.
type FileReader interface {
	StatFile(ctx context.Context, path string) (domain.FileStat, error)
	Read(ctx context.Context, path string) ([]byte, domain.FileStat, error)
	// ReadRange returns host.ErrNotText for a file that is not UTF-8 text.
	ReadRange(ctx context.Context, path string, start, length int) (host.FileRange, error)
}

// FileWriter writes the files of one vault.
type FileWriter interface {
	// Write returns host.ErrStale when the file does not match the baseline.
	Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
}

// Indexer updates the index of one vault for files that changed.
type Indexer interface {
	Scan(ctx context.Context, paths []string) error
}

type NoteLookup interface {
	GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error)
	GetNamedPaths(ctx context.Context, name string) ([]string, error)
	GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error)
}

// LinkLookup finds the links of the notes of one vault.
type LinkLookup interface {
	GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error)
}

// NoteArranger renames, moves and removes notes, and checks the vault for problems.
type NoteArranger interface {
	RenameNote(ctx context.Context, path, title string) (host.NoteRename, error)
	MoveNotes(ctx context.Context, paths []string, folder string) ([]host.NoteMove, error)
	RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error)
	GetProblems(ctx context.Context, checks []string) (host.VaultProblems, error)
}

// LinkEditor writes the links in the frontmatter of notes.
type LinkEditor interface {
	AddLinks(ctx context.Context, additions []host.LinkAddition) ([]host.LinkAdded, error)
	UpdateLinks(ctx context.Context, from string, to domain.Address, change domain.Link, baseline domain.FileStat) (domain.FileStat, error)
	RemoveLinks(ctx context.Context, from string, to domain.Address, role domain.LinkRole, baseline domain.FileStat) (domain.FileStat, error)
}

// VaultSelector names the vault of a call and lists the vaults of the installation.
type VaultSelector interface {
	GetVault(ctx context.Context) (domain.Vault, error)
	GetVaults(ctx context.Context) ([]domain.Vault, error)
}

// VaultList lists the vaults of the installation and changes the list.
type VaultList interface {
	GetVaultList(ctx context.Context) ([]host.VaultEntry, error)
	AddVault(ctx context.Context, root, name string) (domain.Vault, error)
	RenameVault(ctx context.Context, id domain.VaultID, name string) (domain.Vault, error)
	RemoveVault(ctx context.Context, id domain.VaultID) (folder string, err error)
	OpenVault(ctx context.Context, id domain.VaultID) error
}

// VersionLister lists the text versions of a file.
type VersionLister interface {
	ListVersions(ctx context.Context, path string) (host.TextListing, error)
}

// TextReader reads the text of a version of a file.
type TextReader interface {
	ReadText(ctx context.Context, path string, id domain.TextVersionID, cursor string) (host.TextPage, error)
	ReadTextRange(ctx context.Context, path string, id domain.TextVersionID, start, length int) (host.TextRange, error)
}

// PassageSearch searches the passages of the vault index.
type PassageSearch interface {
	SearchPassages(ctx context.Context, query host.PassageQuery) ([]host.Passage, error)
}

// NotePathResolver turns what an agent names a note into the path of its file.
type NotePathResolver interface {
	Resolve(ctx context.Context, name string) string
}
