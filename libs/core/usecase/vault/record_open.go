// Package vault holds the scenarios that act on a vault as a whole: adding one,
// finding one, listing them, synchronizing one with the index.
//
// A scenario orchestrates ports and entities and has no knowledge of a
// filesystem or a database beyond what the domain has.
package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Scenarios covers the vaults an installation holds, the operations on them,
// and the vault's own tree.
type Scenarios struct {
	Registry vaultport.VaultRegistry

	List        List
	RecordOpen  RecordOpen
	Find        Find
	FolderCheck FolderCheck
	Remove      Remove

	Add        Add
	Rename     Rename
	Unregister Unregister
	Erase      Erase

	Move   Move
	Import Import
}

type RecordOpen struct {
	registry vaultport.VaultRegistry
}

func NewRecordOpen(registry vaultport.VaultRegistry) RecordOpen {
	return RecordOpen{registry: registry}
}

func (u RecordOpen) Execute(id domain.VaultID) error {
	return u.registry.RecordOpened(id)
}

// WriteOpenTabs saves the open tabs of a vault.
func WriteOpenTabs(ctx context.Context, store windowport.OpenTabsStore, v domain.Vault, open domain.OpenTabs) error {
	return store.WriteOpenTabs(ctx, v, open)
}

// ReadOpenTabs loads the open tabs of a vault.
func ReadOpenTabs(ctx context.Context, store windowport.OpenTabsStore, v domain.Vault) (domain.OpenTabs, error) {
	return store.ReadOpenTabs(ctx, v)
}
