// Package vault holds the scenarios that act on a vault as a whole: adding one,
// finding one, listing them, synchronizing one with the index.
//
// A scenario orchestrates ports and entities and has no knowledge of a
// filesystem or a database beyond what the domain has.
package vault

import (
	"context"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// Tree imports the files of a vault.
type Tree struct {
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

type RecordOpenByName struct {
	lookup vaultport.VaultLookup
	record RecordOpen
}

// NewRecordOpenByName creates the use case that resolves the vault with find and records it with record.
func NewRecordOpenByName(lookup vaultport.VaultLookup, record RecordOpen) RecordOpenByName {
	return RecordOpenByName{lookup: lookup, record: record}
}

// ExecuteByName resolves the vault and records it as the vault opened last.
func (u RecordOpenByName) ExecuteByName(nameOrPath string) (domain.Vault, error) {
	v, err := vaultdomain.FindByName(nameOrPath, u.lookup.FindVault)
	if err != nil {
		return domain.Vault{}, err
	}
	if err := u.record.Execute(v.ID); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}

type WriteOpenTabs struct {
	store OpenTabsStore
}

func NewWriteOpenTabs(store OpenTabsStore) WriteOpenTabs {
	return WriteOpenTabs{store: store}
}

// Execute replaces the tabs kept for the vault.
func (u WriteOpenTabs) Execute(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error {
	return u.store.WriteOpenTabs(ctx, v, open)
}

type ReadOpenTabs struct {
	store OpenTabsStore
}

func NewReadOpenTabs(store OpenTabsStore) ReadOpenTabs {
	return ReadOpenTabs{store: store}
}

// Execute returns the tabs last written for the vault, and none where nothing was written.
func (u ReadOpenTabs) Execute(ctx context.Context, v domain.Vault) (windowdomain.OpenTabs, error) {
	return u.store.ReadOpenTabs(ctx, v)
}
