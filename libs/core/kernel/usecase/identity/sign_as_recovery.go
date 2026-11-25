package identity

import (
	"context"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

// SignAsRecovery signs a message of a plugin with the recovery key of the pending restore.
type SignAsRecovery struct{ restores restoreHold }

// NewSignAsRecovery returns the use case that signs with the recovery key.
func NewSignAsRecovery(d Deps) *SignAsRecovery { return &SignAsRecovery{restores: d.progress.restores} }

// Execute signs message under the plugin id with the recovery key, only while a restore is pending
// (ErrNoPendingRestore otherwise).
func (u *SignAsRecovery) Execute(_ context.Context, pluginID string, message []byte) ([]byte, error) {
	if pluginID == "" {
		return nil, errNoPluginID
	}
	store, release := u.restores.hold()
	defer release()
	restore := store.current
	if restore == nil {
		return nil, identitydomain.ErrNoPendingRestore
	}
	if restore.refusal != nil {
		return nil, restore.refusal
	}
	signature := restore.recovery.SignForRecovery(pluginID, message)
	return signature[:], nil
}
