// Package vault holds the scenarios that act on a vault as a whole: adding one,
// finding one, listing them, synchronizing one with the index.
//
// A scenario orchestrates ports and entities and has no knowledge of a
// filesystem or a database beyond what the domain has.
package vault

import (
	"context"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/windowport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Scenarios covers the operations on the vaults an installation holds, and the vault's own tree.
type Scenarios struct {
	Change     Changing
	Tree       Tree
	RecordOpen RecordOpen
}

// Changing changes the list of vaults and the vaults on it.
type Changing struct {
	Add        Add
	Rename     Rename
	Remove     Remove
	Unregister Unregister
	Erase      Erase
}

// Tree moves and imports the files of a vault.
type Tree struct {
	Move   Move
	Import Import
}

type RecordOpen struct {
	recorder vaultport.VaultRecorder
}

func NewRecordOpen(recorder vaultport.VaultRecorder) RecordOpen {
	return RecordOpen{recorder: recorder}
}

func (u RecordOpen) Execute(id domain.VaultID) error {
	return u.recorder.RecordOpened(id)
}

// WriteOpenTabs saves the open tabs of a vault.
type WriteOpenTabs struct {
	store windowport.OpenTabsStore
}

// NewWriteOpenTabs is the use case that saves open tabs in the store.
func NewWriteOpenTabs(store windowport.OpenTabsStore) WriteOpenTabs {
	return WriteOpenTabs{store: store}
}

// Execute replaces the tabs kept for the vault.
func (u WriteOpenTabs) Execute(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error {
	return u.store.WriteOpenTabs(ctx, v, open)
}

// ReadOpenTabs loads the open tabs of a vault.
type ReadOpenTabs struct {
	store windowport.OpenTabsStore
}

// NewReadOpenTabs is the use case that loads open tabs from the store.
func NewReadOpenTabs(store windowport.OpenTabsStore) ReadOpenTabs {
	return ReadOpenTabs{store: store}
}

// Execute returns the tabs last written for the vault, and none where nothing was written.
func (u ReadOpenTabs) Execute(ctx context.Context, v domain.Vault) (windowdomain.OpenTabs, error) {
	return u.store.ReadOpenTabs(ctx, v)
}
