// Package vault holds the scenarios that act on a vault as a whole: adding one,
// finding one, listing them, bringing one up to date with the index.
//
// A scenario orchestrates ports and entities and knows no more about a
// filesystem or a database than the domain does.
package vault

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Scenarios is the list of vaults an installation holds and what a person does
// to it, and the vault's own tree as they move things about in it.
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
