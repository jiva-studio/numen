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

type RecordOpenByName struct {
	find   VaultFinding
	record RecordOpen
}

// NewRecordOpenByName creates the use case that resolves the vault with find and records it with record.
func NewRecordOpenByName(find VaultFinding, record RecordOpen) RecordOpenByName {
	return RecordOpenByName{find: find, record: record}
}

// ExecuteByName resolves the vault and records it as the vault opened last.
func (u RecordOpenByName) ExecuteByName(nameOrPath string) (domain.Vault, error) {
	v, err := u.find(nameOrPath)
	if err != nil {
		return domain.Vault{}, err
	}
	if err := u.record.Execute(v.ID); err != nil {
		return domain.Vault{}, err
	}
	return v, nil
}

type WriteOpenTabs struct {
	store windowport.OpenTabsStore
}

func NewWriteOpenTabs(store windowport.OpenTabsStore) WriteOpenTabs {
	return WriteOpenTabs{store: store}
}

// Execute replaces the tabs kept for the vault.
func (u WriteOpenTabs) Execute(ctx context.Context, v domain.Vault, open windowdomain.OpenTabs) error {
	return u.store.WriteOpenTabs(ctx, v, open)
}

type ReadOpenTabs struct {
	store windowport.OpenTabsStore
}

func NewReadOpenTabs(store windowport.OpenTabsStore) ReadOpenTabs {
	return ReadOpenTabs{store: store}
}

// Execute returns the tabs last written for the vault, and none where nothing was written.
func (u ReadOpenTabs) Execute(ctx context.Context, v domain.Vault) (windowdomain.OpenTabs, error) {
	return u.store.ReadOpenTabs(ctx, v)
}
