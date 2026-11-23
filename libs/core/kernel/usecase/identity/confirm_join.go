package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

// ConfirmJoin is the answer of the person on the new device to the account the host offers.
type ConfirmJoin struct{ deps Deps }

// NewConfirmJoin returns the use case that joins the account or declines it.
func NewConfirmJoin(d Deps) *ConfirmJoin { return &ConfirmJoin{deps: d} }

// Execute keeps the chain and then the record when the person approves, and tells the host; a decline keeps nothing.
// It returns ErrNoLink when no account waits for the person, and ErrIdentityExists when an identity appeared on this device meanwhile.
func (u *ConfirmJoin) Execute(ctx context.Context, approve bool) error {
	store := u.deps.links
	store.mu.Lock()
	defer store.mu.Unlock()
	store.expireLocked(u.deps.clock.Now())
	link := store.current
	if link == nil || link.role != identitydomain.LinkRoleJoiner || link.phase != identitydomain.LinkPhaseConfirmAccount {
		return identitydomain.ErrNoLink
	}
	if !approve {
		store.sendLocked(link, encodeRefuse(identitydomain.LinkReasonDeclined, 0))
		store.endLocked(link, identitydomain.LinkPhaseFailed, identitydomain.LinkReasonDeclined)
		return nil
	}
	if err := checkAbsent(ctx, u.deps); err != nil {
		return err
	}
	state, err := identitylib.NewState(identitylib.Record{Device: link.joiner.device, Wrapped: link.joiner.wrapped}, link.joiner.chain)
	if err != nil {
		return err
	}
	if err := u.deps.writeNewState(ctx, state.Record, state.Chain); err != nil {
		return err
	}
	store.sendLocked(link, []byte{messageDone})
	store.setPhaseLocked(link, identitydomain.LinkPhaseLinked)
	return nil
}
