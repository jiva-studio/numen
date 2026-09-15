package vault

import "github.com/jiva-studio/numen/modules/libs/core/port"

// Scenarios is the list of vaults an installation holds and what a person does
// to it, and the vault's own tree as they move things about in it.
type Scenarios struct {
	Registry port.VaultRegistry

	Add    Add
	Rename Rename
	Forget Forget
	Erase  Erase

	Move   Move
	Import Import
}
