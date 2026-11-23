package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// noSyncEntitlements is the source of GetEntitlements until the host holds a session on the sync server: it answers
// identity.ErrNoIdentity where the device has no identity and identity.ErrSyncUnavailable otherwise.
type noSyncEntitlements struct {
	account hostendpoint.Account
}

func (s noSyncEntitlements) GetEntitlements(ctx context.Context) ([]identitydomain.Entitlement, error) {
	if _, err := s.account.GetPublicIdentity(ctx); err != nil {
		return nil, err
	}
	return nil, identitydomain.ErrSyncUnavailable
}
