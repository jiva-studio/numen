package registry

import (
	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// FolderCheck reports whether a vault's folder is not where the list records it.
// The list records where a vault was last seen and the vault carries the
// identity, so a folder that has gone leaves the vault on the list, marked.
// Every screen that shows the list uses this one query.
type FolderCheck struct {
	readers vaultport.VaultReaders
}

// NewFolderCheck returns a FolderCheck that opens the folder as a vault.
func NewFolderCheck(readers vaultport.VaultReaders) FolderCheck {
	return FolderCheck{readers: readers}
}

// Execute reports whether the folder is gone. A path holding something that is
// not a folder is gone as well.
func (u FolderCheck) Execute(v domain.Vault) bool {
	_, err := u.readers.Open(v)
	return err != nil
}
