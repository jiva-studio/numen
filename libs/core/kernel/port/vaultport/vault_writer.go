package vaultport

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type NoteWriter interface {
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
}

type FolderWriter interface {
	MakeFolder(ctx context.Context, path string) error
	Import(ctx context.Context, path string, content io.Reader) error
}

type FileMover interface {
	Move(ctx context.Context, from, to string) error
	// CheckMoveTarget returns vault.ErrAlreadyExists when a file other than the one at from is at to. On
	// a filesystem that ignores case or Unicode normalization, a path that differs from from only in
	// case or normalization is the same file.
	CheckMoveTarget(ctx context.Context, from, to string) error
	Remove(ctx context.Context, path string) error
}

// TrashEmptier deletes what a vault keeps in its trash.
type TrashEmptier interface {
	// EmptyTrash deletes everything under the vault's trash and returns the paths it could not delete.
	EmptyTrash(ctx context.Context) (undeleted []string, err error)
}

// NoteWriters opens a vault for writing notes. Which vault a use case works on
// is decided while it runs.
type NoteWriters interface {
	OpenNoteWriter(v domain.Vault) (NoteWriter, error)
}

// FolderWriters opens a vault for filling its folders.
type FolderWriters interface {
	OpenFolderWriter(v domain.Vault) (FolderWriter, error)
}

// FileMovers opens a vault for moving and removing its files.
type FileMovers interface {
	OpenFileMover(v domain.Vault) (FileMover, error)
}

// TrashEmptiers opens a vault for emptying its trash.
type TrashEmptiers interface {
	OpenTrashEmptier(v domain.Vault) (TrashEmptier, error)
}

// VaultLocks holds a vault's write lock, so two writers do not change a vault at once.
type VaultLocks interface {
	// Lock waits while another holder has the lock and returns what releases it.
	Lock(ctx context.Context, v domain.Vault) (release func(), err error)
}
