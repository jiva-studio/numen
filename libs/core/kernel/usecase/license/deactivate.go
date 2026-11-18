package license

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
)

// Deactivate releases this machine's activation slot at the provider and removes the key from the
// key store once the provider confirms.
type Deactivate struct {
	operation
}

// NewDeactivate returns the use case that deactivates a license on this machine.
func NewDeactivate(deps Deps) *Deactivate {
	return &Deactivate{operation: deps.base}
}

// Execute deactivates the license on this machine and returns the resulting inactive license info.
// The stored envelope is kept if the provider cannot be reached, so the machine instance ID is not
// lost while the slot on the server remains occupied.
func (u *Deactivate) Execute(ctx context.Context, licenseID string) (license.Info, error) {
	decl, err := u.findDeclaration(licenseID)
	if err != nil {
		return license.Info{}, err
	}
	u.gate.Lock()
	env, isFound, err := u.readEnvelope(ctx, decl.ID)
	u.gate.Unlock()
	if err != nil {
		return license.Info{}, err
	}
	if !isFound {
		u.gate.Lock()
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		return info, buildErr
	}
	fingerprint := ""
	if u.machine != nil {
		fingerprint = u.machine.GetMachineFingerprint()
	}
	if env.MachineFingerprint == fingerprint &&
		!decl.IsCustom() &&
		env.Provider != "" &&
		env.Provider != license.ProviderCustom &&
		env.InstanceID != "" {
		if u.validator == nil {
			return license.Info{}, license.ErrProviderUnavailable
		}
		if err := u.validator.Deactivate(ctx, env.Provider, env.Key, env.InstanceID); err != nil && !errors.Is(err, license.ErrNotFound) {
			return license.Info{}, err
		}
	}

	u.gate.Lock()
	if err := u.deleteEnvelope(ctx, decl.ID); err != nil {
		u.gate.Unlock()
		return license.Info{}, err
	}
	info, _, err := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	if err != nil {
		return license.Info{}, err
	}
	u.notifyChange(ctx)
	return info, nil
}
