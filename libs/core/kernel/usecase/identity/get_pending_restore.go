package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// GetPendingRestore reads the public keys of the pending restore.
type GetPendingRestore struct{ restores restoreHold }

// NewGetPendingRestore reads the pending restore held in d, shared with the windows and plugins of the same installation.
func NewGetPendingRestore(d Deps) *GetPendingRestore {
	return &GetPendingRestore{restores: d.progress.restores}
}

// Execute returns the public keys of the pending restore, or a value with IsPending false.
func (u *GetPendingRestore) Execute(context.Context) (identitydomain.PendingRestore, error) {
	store, release := u.restores.hold()
	defer release()
	restore := store.current
	if restore == nil || restore.refusal != nil {
		return identitydomain.PendingRestore{}, nil
	}
	return identitydomain.PendingRestore{
		IsPending:     true,
		RootSignKey:   restore.recovery.Public().Sign,
		DeviceSignKey: restore.device.Public().Sign,
	}, nil
}
