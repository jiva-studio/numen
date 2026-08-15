package appstate

import "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"

// file is the shape on disk. It carries a version because a human may have to
// read it years from now, and a file that does not say what it is is a file
// nobody dares change.
type file struct {
	V      int            `json:"v"`
	Vaults []domain.Vault `json:"vaults"`
}
