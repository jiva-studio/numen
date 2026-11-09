// Package vaultcontract re-exports the vault port contract.
package vaultcontract

import (
	"context"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type (
	IndexClaim      = vaultport.IndexClaim
	VaultIdentity   = vaultport.VaultIdentity
	VaultReader     = vaultport.VaultReader
	VaultReaders    = vaultport.VaultReaders
	VaultRegistry   = vaultport.VaultRegistry
	VaultRepository = vaultport.VaultRepository
	VaultWatcher    = vaultport.VaultWatcher
	VaultWriter     = vaultport.VaultWriter
	VaultWriters    = vaultport.VaultWriters
)

var (
	ErrAlreadyExists   = vaultport.ErrAlreadyExists
	ErrClaimed         = vaultport.ErrClaimed
	ErrExist           = vaultport.ErrExist
	ErrLockedByAnother = vaultport.ErrLockedByAnother
	ErrNotANote        = vaultport.ErrNotANote
	ErrNotExist        = vaultport.ErrNotExist
	ErrOutside         = vaultport.ErrOutside
	ErrStale           = vaultport.ErrStale
)

func IsNoNote(err error) bool { return vaultport.IsNoNote(err) }

func IsNote(path string) bool { return vaultport.IsNote(path) }

func ReadNoteType(ctx context.Context, readers vaultport.VaultReaders, v domain.Vault, path string) (domain.NoteType, error) {
	return vaultport.ReadNoteType(ctx, readers, v, path)
}
