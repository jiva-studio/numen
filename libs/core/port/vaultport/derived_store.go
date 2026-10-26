package vaultport

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrClaimed is what a name another caller holds gets. Work whose whole output
// is one file reads it as that work already being under way.
var ErrClaimed = errors.New("the name is claimed by another caller")

// ErrHeldByAnother is what a name gets when a program outside this one has the
// file open and the system will not open it twice. A synchroniser and a backup
// reader each hold a file for moments at a time, so it is a state to come back
// to and not a failure of the store.
var ErrHeldByAnother = errors.New("the file is held by another program")

// DerivedStore is the application's own folder inside a vault, where the work
// one run is doing on the vault is claimed so no other run does it at once.
type DerivedStore interface {
	// ClaimArtifact takes a name for this caller alone and returns what lets it
	// go. A name another caller holds gets ErrClaimed.
	ClaimArtifact(ctx context.Context, name domain.ArtifactName) (release func() error, err error)
}

// DerivedStores opens one vault's store. Which vault a use case works on is
// decided while it runs, as with readers and writers.
type DerivedStores interface {
	Open(v domain.Vault) (DerivedStore, error)
}
