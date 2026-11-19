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
	result, err, _ := u.flights.Do("activate:"+req.LicenseID+":"+cleanedKey, func() (any, error) {
		return u.activateOnce(ctx, req, cleanedKey)
	})
	if err != nil {
		return license.Info{}, err
	}
	info, ok := result.(license.Info)
	if !ok {
		return license.Info{}, fmt.Errorf("activate license: unexpected result")
	}
	return info, nil
}

func (u *Activate) activateOnce(ctx context.Context, req ActivateRequest, cleanedKey string) (license.Info, error) {
	decl, err := u.findDeclaration(req.LicenseID)
	if err != nil {
		return license.Info{}, err
	}
	fingerprint, instanceName := u.getMachineTarget()
	now := u.clock.Now().UTC()
	nowStr := license.FormatTime(now)

	if decl.IsCustom() {
		return u.writeAndBuild(ctx, decl, license.Envelope{
			Key:                cleanedKey,
			KeyHint:            license.GetKeyHint(cleanedKey),
			Provider:           license.ProviderCustom,
			InstanceName:       instanceName,
			MachineFingerprint: fingerprint,
			Status:             license.StatusActive,
			VerifiedAt:         nowStr,
			LastSeenAt:         nowStr,
		})
	}
	if u.validator == nil {
		return license.Info{}, license.ErrProviderUnavailable
	}
	if info, ok, valErr := u.tryRevalidateSameKey(ctx, decl, cleanedKey, fingerprint, now); ok || valErr != nil {
		return info, valErr
	}
	act, err := u.validator.Activate(ctx, decl.Providers, cleanedKey, instanceName)
	if err != nil {
		return license.Info{}, err
	}
	if act.Status != license.StatusActive {
		if act.Status == license.StatusExpired {
			return license.Info{}, license.ErrExpired
		}
		return license.Info{}, license.ErrDisabled
	}
	env := buildActivatedEnvelope(cleanedKey, instanceName, fingerprint, nowStr, act)
	info, writeErr := u.writeAndBuild(ctx, decl, env)
	if writeErr == nil {
		return info, nil
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	deactivateErr := u.validator.Deactivate(cleanupCtx, env.Provider, env.Key, env.InstanceID)
	cancel()
	if deactivateErr != nil && !errors.Is(deactivateErr, license.ErrNotFound) {
		return license.Info{}, errors.Join(writeErr, deactivateErr)
	}
	return license.Info{}, writeErr
}

func (u *Activate) getMachineTarget() (string, string) {
	if u.machine == nil {
		return "", "numen"
	}
	instanceName := u.machine.GetInstanceName()
	if instanceName == "" {
		instanceName = "numen"
	}
	return u.machine.GetMachineFingerprint(), instanceName
}

func (u *Activate) tryRevalidateSameKey(
	ctx context.Context,
	decl license.Declaration,
	cleanedKey, fingerprint string,
	now time.Time,
) (license.Info, bool, error) {
	u.gate.Lock()
	existing, hasExisting, readErr := u.readEnvelope(ctx, decl.ID)
	u.gate.Unlock()
	if readErr != nil {
		return license.Info{}, false, readErr
	}
	if !isSameMachineActivation(existing, hasExisting, fingerprint, cleanedKey) {
		return license.Info{}, false, nil
	}
	return u.revalidateExisting(ctx, decl, existing, now)
}

func isSameMachineActivation(existing license.Envelope, hasExisting bool, fingerprint, cleanedKey string) bool {
	return hasExisting &&
		existing.MachineFingerprint == fingerprint &&
		license.NormalizeKey(existing.Key) == cleanedKey &&
		existing.InstanceID != "" &&
		existing.Provider != "" &&
		existing.Provider != license.ProviderCustom
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
	current, found, readErr := u.readEnvelope(ctx, decl.ID)
	if readErr != nil {
		u.gate.Unlock()
		return license.Info{}, false, readErr
	}
	if !found || !isSameActivation(current, existing) {
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		return info, true, buildErr
	}
	if writeErr := u.writeEnvelope(ctx, decl.ID, existing); writeErr != nil {
		u.gate.Unlock()
		return license.Info{}, false, writeErr
	}
	info, _, buildErr := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	if buildErr != nil {
		return license.Info{}, false, buildErr
	}
	u.notifyChange(ctx)
	return info, true, nil
}

func buildActivatedEnvelope(
	cleanedKey, instanceName, fingerprint, nowStr string,
	act licenseport.ActivationResult,
) license.Envelope {
	expiresStr := ""
	if act.ExpiresAt != nil {
		expiresStr = license.FormatTime(*act.ExpiresAt)
	}
	return license.Envelope{
		Key:                cleanedKey,
		KeyHint:            license.GetKeyHint(cleanedKey),
		Provider:           act.Provider,
		InstanceID:         act.InstanceID,
		InstanceName:       instanceName,
		MachineFingerprint: fingerprint,
		Status:             act.Status,
		ExpiresAt:          expiresStr,
		VerifiedAt:         nowStr,
		LastSeenAt:         nowStr,
	}
}

func (u *Activate) writeAndBuild(
	ctx context.Context,
	decl license.Declaration,
	env license.Envelope,
) (license.Info, error) {
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
