package vaultport

import (
	"context"
	"errors"
)

// ErrClaimed is what a name another caller holds gets. Work whose whole output
// is one file reads it as that work already being under way.
var ErrClaimed = errors.New("the name is claimed by another caller")

// ErrHeldByAnother is what a name gets when a program outside this one has the
// file open and the system will not open it twice. A synchroniser and a backup
// reader each hold a file for moments at a time, so it is a state to come back
// to and not a failure of the store.
var ErrHeldByAnother = errors.New("the file is held by another program")

// IndexClaim is the hold on one vault's vectors that keeps two processes from
// making them at once.
type IndexClaim interface {
	// ClaimIndex takes the hold for this caller alone and returns what lets it
	// go. A hold another caller has gets ErrClaimed.
	ClaimIndex(ctx context.Context) (release func() error, err error)
}
