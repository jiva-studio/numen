package appstate

import "github.com/jiva-studio/numen/modules/libs/core/domain"

// registryFile is the shape on disk. It carries a version because a human may
// have to read it years from now, and a file that does not say what it is is
// a file nobody dares change.
type registryFile struct {
	V int `json:"v"`
	// Last is the identity of the vault opened most recently, and is absent
	// until one has been opened.
	Last   string         `json:"last,omitempty"`
	Vaults []domain.Vault `json:"vaults"`
}
