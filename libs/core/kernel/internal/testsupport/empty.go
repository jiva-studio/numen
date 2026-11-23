package testsupport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// NoDetection is the detection of a machine where no provider detects connections.
type NoDetection struct{}

func (NoDetection) ListDetected() []connection.Connection { return nil }
func (NoDetection) IsPending(string) bool                 { return false }
func (NoDetection) Detect(context.Context, string)        {}
func (NoDetection) Refresh()                              {}

// EmptyNoteQueries is an index that holds no vault, no note and no name.
type EmptyNoteQueries struct{}

func (EmptyNoteQueries) ListNamedPaths(context.Context, domain.VaultID, string) ([]string, error) {
	return nil, nil
}

func (EmptyNoteQueries) GetNotes(context.Context, domain.VaultID, []string) (map[string]domain.NoteRef, error) {
	return nil, nil
}

func (EmptyNoteQueries) Has(context.Context, domain.VaultID) (bool, error) { return false, nil }

// EmptySourceBacklog is the backlog of an index that has no source to cut again.
type EmptySourceBacklog struct{}

func (EmptySourceBacklog) GetUnchunkedSources(context.Context, domain.VaultID, domain.SourceKind, int) ([]string, error) {
	return nil, nil
}

func (EmptySourceBacklog) GetSourcesWithOtherRecipe(context.Context, domain.VaultID, domain.SourceKind, []string, int) ([]string, error) {
	return nil, nil
}

// ErrVaultNotOpen is what ClosedVaultReaders answers for every vault.
var ErrVaultNotOpen = errors.New("this vault's files are not open")

// ClosedVaultReaders opens no vault.
type ClosedVaultReaders struct{}

func (ClosedVaultReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	return nil, ErrVaultNotOpen
}

// FreeIndexClaim holds nothing and is never busy.
type FreeIndexClaim struct{}

func (FreeIndexClaim) ClaimIndex(context.Context) (func() error, error) {
	return func() error { return nil }, nil
}
