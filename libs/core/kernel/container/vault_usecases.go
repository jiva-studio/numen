package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"

// vaultUseCases covers the operations on the vaults an installation holds, and the vault's own tree.
type vaultUseCases struct {
	Change vaultListUseCases
	Tree   vault.Tree
}

// vaultListUseCases changes the list of vaults and the vaults on it.
type vaultListUseCases struct {
	Add        vault.Add
	Rename     vault.Rename
	Remove     vault.Remove
	Unregister vault.Unregister
	Erase      vault.Erase
}
