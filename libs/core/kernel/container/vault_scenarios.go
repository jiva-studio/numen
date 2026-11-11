package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"

// VaultScenarios covers the operations on the vaults an installation holds, and the vault's own tree.
type VaultScenarios struct {
	Change     VaultChanging
	Tree       vault.Tree
	RecordOpen vault.RecordOpen
}

// VaultChanging changes the list of vaults and the vaults on it.
type VaultChanging struct {
	Add        vault.Add
	Rename     vault.Rename
	Remove     vault.Remove
	Unregister vault.Unregister
	Erase      vault.Erase
}
