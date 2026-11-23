package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetPendingRestore reads the public keys of the pending restore.
type GetPendingRestore struct{ deps Deps }

// NewGetPendingRestore reads the pending restore held in d, shared with the windows and plugins of the same installation.
func NewGetPendingRestore(d Deps) *GetPendingRestore { return &GetPendingRestore{deps: d} }

// Execute returns the public keys of the pending restore, or a value with IsPending false.
func (u *GetPendingRestore) Execute(context.Context) (identitydomain.PendingRestore, error) {
	restore := u.deps.pending.read()
	if restore == nil {
		return identitydomain.PendingRestore{}, nil
	}
	return identitydomain.PendingRestore{
		IsPending:     true,
		RootSignKey:   restore.recovery.Public().Sign,
		DeviceSignKey: restore.device.Public().Sign,
	}, nil
}
