package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// ConfirmJoin is the answer of the person on the new device to the account the host offers.
type ConfirmJoin struct {
	links   linkHold
	clock   machineport.InstallationClock
	records records
}

// NewConfirmJoin returns the use case that joins the account or declines it.
func NewConfirmJoin(d Deps) *ConfirmJoin {
	return &ConfirmJoin{links: d.progress.links, clock: d.clock, records: d.records}
}

// Execute keeps the chain and then the record when the person approves, and tells the host; a decline keeps nothing.
// It returns ErrNoLink when no account waits for the person, and ErrIdentityExists when an identity appeared on this device meanwhile.
func (u *ConfirmJoin) Execute(ctx context.Context, approve bool) error {
	store, release := u.links.hold()
	defer release()
	store.expireLocked(u.clock.Now())
	link := store.current
	if link == nil || link.progress.Role != identitydomain.LinkRoleJoiner || link.progress.Phase != identitydomain.LinkPhaseConfirmAccount {
		return identitydomain.ErrNoLink
	}
	if !approve {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonDeclined))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonDeclined)
		return nil
	}
	if err := u.records.checkAbsent(ctx); err != nil {
		return err
	}
	state, err := identitylib.NewState(link.record, link.chain)
	if err != nil {
		return err
	}
	if err := u.records.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return err
	}
	store.sendLocked(link, []byte{messageDone})
	store.setPhaseLocked(link, identitydomain.LinkPhaseLinked)
	return nil
}
