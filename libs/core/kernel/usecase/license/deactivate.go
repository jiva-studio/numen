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
	if err := u.deactivateRemoteIfNeeded(ctx, decl, env); err != nil {
		return license.Info{}, err
	}
	return u.deleteAndBuild(ctx, decl)
}

func (u *Deactivate) deactivateRemoteIfNeeded(
	ctx context.Context,
	decl license.Declaration,
	env license.Envelope,
) error {
	fingerprint := ""
	if u.machine != nil {
		fingerprint = u.machine.GetMachineFingerprint()
	}
	if !hasRemoteInstance(decl, env, fingerprint) {
		return nil
	}
	if u.validator == nil {
		return license.ErrProviderUnavailable
	}
	err := u.validator.Deactivate(ctx, env.Provider, env.Key, env.InstanceID)
	if err != nil && !errors.Is(err, license.ErrNotFound) {
		return err
	}
	return nil
}

func hasRemoteInstance(decl license.Declaration, env license.Envelope, fingerprint string) bool {
	return env.MachineFingerprint == fingerprint &&
		!decl.IsCustom() &&
		env.Provider != "" &&
		env.Provider != license.ProviderCustom &&
		env.InstanceID != ""
}

func (u *Deactivate) deleteAndBuild(ctx context.Context, decl license.Declaration) (license.Info, error) {
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
