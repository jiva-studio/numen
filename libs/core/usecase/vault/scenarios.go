package vault

import (
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
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
