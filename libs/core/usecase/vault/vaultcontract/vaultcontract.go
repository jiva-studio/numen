// Package vaultcontract re-exports the vault port contract.
package vaultcontract

import "github.com/jiva-studio/numen/libs/core/port/vaultport"

type (
	IndexClaim = vaultport.IndexClaim
	VaultIdentity = vaultport.VaultIdentity
	VaultReader = vaultport.VaultReader
	VaultReaders = vaultport.VaultReaders
	VaultRegistry = vaultport.VaultRegistry
	VaultRepository = vaultport.VaultRepository
	VaultWatcher = vaultport.VaultWatcher
	VaultWriter = vaultport.VaultWriter
	VaultWriters = vaultport.VaultWriters
)

var (
	ErrAlreadyExists = vaultport.ErrAlreadyExists
	ErrClaimed = vaultport.ErrClaimed
	ErrExist = vaultport.ErrExist
	ErrHeldByAnother = vaultport.ErrHeldByAnother
	ErrNotANote = vaultport.ErrNotANote
	ErrNotExist = vaultport.ErrNotExist
	ErrOutside = vaultport.ErrOutside
	ErrStale = vaultport.ErrStale
	IsNoNote = vaultport.IsNoNote
	IsNote = vaultport.IsNote
	ReadNoteType = vaultport.ReadNoteType
)
