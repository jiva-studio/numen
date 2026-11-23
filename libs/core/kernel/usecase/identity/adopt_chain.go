package identity

import (
	"context"
	"fmt"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// AdoptChain takes a chain the server holds.
type AdoptChain struct {
	links   linkHold
	records records
}

// NewAdoptChain returns the use case that adopts a chain.
func NewAdoptChain(d Deps) *AdoptChain {
	return &AdoptChain{links: d.progress.links, records: d.records}
}

// Execute takes the chain when it is the chain of this device, a longer one that extends it, or one that a recovery-signed entry makes win at a
// fork, which voids the entries of the devices after it (identitylib.ErrNotExtension otherwise; identitylib.ErrDeviceVoided when this device
// was added by a voided entry).
// When the epoch grew the record takes the keys the chain wraps for this device; when the chain revokes this device the record stays as it is
// and the device is revoked. Taking the same chain again changes nothing.
func (u *AdoptChain) Execute(ctx context.Context, chainBytes []byte) error {
	chain, err := identitylib.ParseChain(chainBytes)
	if err != nil {
		return fmt.Errorf("%w: %w", identitydomain.ErrInvalidChain, err)
	}
	_, release := u.links.hold()
	defer release()
	state, err := u.records.readState(ctx)
	if err != nil {
		return err
	}
	_, err = u.records.adoptState(ctx, state, chain)
	return err
}
