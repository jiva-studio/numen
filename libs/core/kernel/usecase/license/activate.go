package license

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/core/kernel/port/licenseport"
)

// ActivateRequest is the input for activating a license key on this machine.
type ActivateRequest struct {
	LicenseID string
	Key       string
}

// Activate registers this machine with one of the license's providers (or stores a custom key) and
// writes the resulting envelope to the key store.
type Activate struct {
	operation
}

// NewActivate returns the use case that activates a license key on this machine.
func NewActivate(deps Deps) *Activate {
	return &Activate{operation: deps.base}
}

// Execute activates the license key and returns the updated license info.
func (u *Activate) Execute(ctx context.Context, req ActivateRequest) (license.Info, error) {
	cleanedKey := license.NormalizeKey(req.Key)
	if cleanedKey == "" {
		return license.Info{}, fmt.Errorf("%w: license key is required", license.ErrInvalidKey)
	}
	decl, err := u.findDeclaration(req.LicenseID)
	if err != nil {
		return license.Info{}, err
	}
	fingerprint := ""
	instanceName := "numen"
	if u.machine != nil {
		fingerprint = u.machine.GetMachineFingerprint()
		if name := u.machine.GetInstanceName(); name != "" {
			instanceName = name
		}
	}
	now := u.clock.Now().UTC()
	nowStr := license.FormatTime(now)

	if decl.IsCustom() {
		return u.saveCustomEnvelope(ctx, decl, cleanedKey, instanceName, fingerprint, nowStr)
	}
	if u.validator == nil {
		return license.Info{}, license.ErrProviderUnavailable
	}

	u.gate.Lock()
	existing, hasExisting, readErr := u.readEnvelope(ctx, decl.ID)
	u.gate.Unlock()
	if readErr != nil {
		return license.Info{}, readErr
	}

	// Re-entering the same key on the same machine revalidates the existing instance ID so no extra
	// activation slot is burned.
	if hasExisting &&
		existing.MachineFingerprint == fingerprint &&
		license.NormalizeKey(existing.Key) == cleanedKey &&
		existing.InstanceID != "" &&
		existing.Provider != "" &&
		existing.Provider != license.ProviderCustom {
		if info, ok, valErr := u.revalidateExisting(ctx, decl, existing, now); ok || valErr != nil {
			return info, valErr
		}
	}

	act, err := u.validator.Activate(ctx, decl.Providers, cleanedKey, instanceName)
	if err != nil {
		return license.Info{}, err
	}
	return u.saveActivatedEnvelope(ctx, decl, cleanedKey, instanceName, fingerprint, nowStr, act)
}

func (u *Activate) saveCustomEnvelope(
	ctx context.Context,
	decl license.Declaration,
	cleanedKey, instanceName, fingerprint, nowStr string,
) (license.Info, error) {
	u.gate.Lock()
	env := license.Envelope{
		Key:                cleanedKey,
		KeyHint:            license.GetKeyHint(cleanedKey),
		Provider:           license.ProviderCustom,
		InstanceName:       instanceName,
		MachineFingerprint: fingerprint,
		Status:             license.StatusActive,
		VerifiedAt:         nowStr,
		LastSeenAt:         nowStr,
	}
	if err := u.writeEnvelope(ctx, decl.ID, env); err != nil {
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

func (u *Activate) revalidateExisting(
	ctx context.Context,
	decl license.Declaration,
	existing license.Envelope,
	now time.Time,
) (license.Info, bool, error) {
	val, err := u.validator.Validate(ctx, existing.Provider, existing.Key, existing.InstanceID)
	if err != nil {
		if errors.Is(err, license.ErrNotFound) {
			return license.Info{}, false, nil
		}
		return license.Info{}, false, err
	}
	if val.Status != license.StatusActive {
		if val.Status == license.StatusExpired {
			return license.Info{}, false, license.ErrExpired
		}
		return license.Info{}, false, license.ErrDisabled
	}
	nowStr := license.FormatTime(now)
	existing.Status = license.StatusActive
	existing.VerifiedAt = nowStr
	existing.LastSeenAt = nowStr
	if val.ExpiresAt != nil {
		existing.ExpiresAt = license.FormatTime(*val.ExpiresAt)
	}
	u.gate.Lock()
	if err := u.writeEnvelope(ctx, decl.ID, existing); err != nil {
		u.gate.Unlock()
		return license.Info{}, false, err
	}
	info, _, err := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	if err != nil {
		return license.Info{}, false, err
	}
	u.notifyChange(ctx)
	return info, true, nil
}

func (u *Activate) saveActivatedEnvelope(
	ctx context.Context,
	decl license.Declaration,
	cleanedKey, instanceName, fingerprint, nowStr string,
	act licenseport.ActivationResult,
) (license.Info, error) {
	expiresStr := ""
	if act.ExpiresAt != nil {
		expiresStr = license.FormatTime(*act.ExpiresAt)
	}
	env := license.Envelope{
		Key:                cleanedKey,
		KeyHint:            license.GetKeyHint(cleanedKey),
		Provider:           act.Provider,
		InstanceID:         act.InstanceID,
		InstanceName:       instanceName,
		MachineFingerprint: fingerprint,
		Status:             license.StatusActive,
		ExpiresAt:          expiresStr,
		VerifiedAt:         nowStr,
		LastSeenAt:         nowStr,
	}
	u.gate.Lock()
	if err := u.writeEnvelope(ctx, decl.ID, env); err != nil {
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
