package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ScanVault reads the whole vault the user names and reports how much of it the index holds afterwards.
type ScanVault struct {
	read      ReadWholeVault
	summaries noteport.VaultSummaries
}

// ScanVaultResult is the vault that was read, what each pass made, and what the index holds of the vault.
type ScanVaultResult struct {
	Vault   domain.Vault
	Made    ReadWholeVaultResult
	Summary domain.VaultSummary
}

// NewScanVault returns the use case over a read that finds its vault by name.
func NewScanVault(read ReadWholeVault, summaries noteport.VaultSummaries) ScanVault {
	return ScanVault{read: read, summaries: summaries}
}

// Execute reads the vault named by nameOrPath and counts what the index holds of it.
func (u ScanVault) Execute(ctx context.Context, nameOrPath string, reporting Reporting) (ScanVaultResult, error) {
	v, made, err := u.read.ExecuteByName(ctx, nameOrPath, reporting)
	if err != nil {
		return ScanVaultResult{}, err
	}
	summary, err := u.summaries.GetSummary(ctx, v.ID)
	if err != nil {
		return ScanVaultResult{}, err
	}
	return ScanVaultResult{Vault: v, Made: made, Summary: summary}, nil
}
