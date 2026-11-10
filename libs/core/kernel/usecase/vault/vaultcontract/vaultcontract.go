// Package vaultcontract re-exports the vault port contract.
package vaultcontract

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

type (
	IndexClaim      = vaultport.IndexClaim
	VaultIdentity   = vaultport.VaultIdentity
	VaultReader     = vaultport.VaultReader
	VaultReaders    = vaultport.VaultReaders
	VaultRepository = vaultport.VaultRepository
	VaultWatcher    = vaultport.VaultWatcher
)

var (
	ErrAlreadyExists   = vault.ErrAlreadyExists
	ErrClaimed         = vault.ErrClaimed
	ErrExist           = vaultport.ErrExist
	ErrLockedByAnother = vault.ErrLockedByAnother
	ErrNotANote        = vaultport.ErrNotANote
	ErrNotExist        = vaultport.ErrNotExist
	ErrOutside         = vault.ErrOutside
	ErrStale           = vaultport.ErrStale
)

func IsNoNote(err error) bool { return vaultport.IsNoNote(err) }
