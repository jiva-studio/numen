package vaultport

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/edit"
)

// ErrStale is the write error returned when the file it was about to replace is
// no longer the one the caller read.
var ErrStale = edit.ErrStale

// ErrAlreadyExists is the move error returned when something already exists
// where a note was to go.
var ErrAlreadyExists = errors.New("a file is already there")

type vaultPathWriter interface {
	Move(ctx context.Context, from, to string) error
	MakeFolder(ctx context.Context, path string) error
	Remove(ctx context.Context, path string) error
}

// VaultWriter changes the files of one vault.
type VaultWriter interface {
	vaultPathWriter
	Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error)
	Create(ctx context.Context, path string, content []byte) error
	Import(ctx context.Context, path string, content io.Reader) error
}

// VaultWriters opens a vault for writing.
type VaultWriters interface {
	Open(v domain.Vault) (VaultWriter, error)
	Lock(ctx context.Context, v domain.Vault) (release func(), err error)
}
