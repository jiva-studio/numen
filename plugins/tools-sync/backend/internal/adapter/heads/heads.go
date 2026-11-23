// Package heads lists the heads of the live notes of a synced vault the way a pull verifies them.
package heads

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

var errNotImplemented = errors.New("not implemented")

// Deps are what the adapter needs.
type Deps struct {
	Catalog   ports.Catalog
	Chain     ports.Chain
	Revisions ports.Revisions
	Crypto    ports.Crypto
	Identity  ports.Identity
}

// Heads reads the heads of a synced vault from the server.
type Heads struct{ deps Deps }

// New returns the adapter over the dependencies.
func New(deps Deps) *Heads { return &Heads{deps: deps} }

// ListHeads reads every page from the first revision, checks each revision like a pull and returns the live ones.
func (h *Heads) ListHeads(_ context.Context, _ string) ([]ports.Head, error) {
	return nil, errNotImplemented
}
