package appstate

import "github.com/jiva-studio/numen/libs/note/domain"

// registryFile is the on-disk shape. It carries a version because it may need to be
// read years later, and a file that does not identify its format cannot be changed
// safely.
type registryFile struct {
	Version int            `json:"v"`
	LastID  domain.VaultID `json:"last,omitempty"`
	Vaults  []vaultEntry   `json:"vaults"`
}

// vaultEntry is the on-disk JSON representation of one registered vault.
type vaultEntry struct {
	ID   domain.VaultID `json:"id"`
	Name string         `json:"name"`
	Path string         `json:"path"`
}
