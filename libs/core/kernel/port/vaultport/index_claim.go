package vaultport

import (
	"context"
)

// IndexClaim is the hold on one vault's vectors that keeps two processes from
// making them at once.
type IndexClaim interface {
	// ClaimIndex takes the hold for this caller alone and returns what lets it
	// go. A hold another caller has gets vault.ErrClaimed.
	ClaimIndex(ctx context.Context) (release func() error, err error)
}
