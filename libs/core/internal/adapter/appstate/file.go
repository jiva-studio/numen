package appstate

import "github.com/jiva-studio/numen/libs/notes/domain"

// registryFile is the on-disk shape. It carries a version because it may need to be
// read years later, and a file that does not identify its format cannot be changed
// safely.
type registryFile struct {
	Version int `json:"v"`
	// LastID is the identity of the vault opened most recently, and is absent
	// until one has been opened.
	LastID domain.VaultID `json:"last,omitempty"`
	Vaults []domain.Vault `json:"vaults"`
}
