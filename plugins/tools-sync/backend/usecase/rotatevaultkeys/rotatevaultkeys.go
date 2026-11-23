// Package rotatevaultkeys gives every connected vault a key of the epoch of the account keys.
package rotatevaultkeys

import (
	"context"
	"errors"
	"io"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the use case needs.
type Deps struct {
	Catalog  ports.Catalog
	Chain    ports.Chain
	Crypto   ports.Crypto
	Identity ports.Identity
	Host     ports.Host
	State    ports.Links
	Random   io.Reader
}

// RotateVaultKeys appends the catalog entries that carry a new vault key for the epoch of the account.
type RotateVaultKeys struct{ deps Deps }

// NewRotateVaultKeys returns the use case over the dependencies.
func NewRotateVaultKeys(deps Deps) *RotateVaultKeys { return &RotateVaultKeys{deps: deps} }

// Execute returns the number of vaults it gave a new key.
func (u *RotateVaultKeys) Execute(ctx context.Context) (int, error) {
	return 0, errors.New("not implemented")
}
