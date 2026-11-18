package license

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
)

// Verify checks a stored license online with its provider when its freshness TTL has elapsed or
// its status requires online verification, coalescing concurrent checks per license ID.
type Verify struct {
	operation
}

// NewVerify returns the use case that verifies a stored license online.
func NewVerify(deps Deps) *Verify {
	return &Verify{operation: deps.base}
}

// Execute verifies the license with its provider and updates the stored envelope when the status
// or timestamps change.
func (u *Verify) Execute(ctx context.Context, licenseID string) (license.Info, error) {
	res, err, _ := u.flights.Do(licenseID, func() (any, error) {
		return u.verifyOnce(ctx, licenseID)
	})
	if err != nil {
		return license.Info{}, err
	}
	info, _ := res.(license.Info)
	return info, nil
}

func (u *Verify) verifyOnce(ctx context.Context, licenseID string) (license.Info, error) {
	decl, err := u.findDeclaration(licenseID)
	if err != nil {
		return license.Info{}, err
	}
	u.gate.Lock()
	env, isFound, err := u.readEnvelope(ctx, decl.ID)
	if err != nil || !isFound {
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		if err != nil {
			return license.Info{}, err
		}
		return info, buildErr
	}
	fingerprint := ""
	if u.machine != nil {
		fingerprint = u.machine.GetMachineFingerprint()
	}
	now := u.clock.Now().UTC()
	eval := license.EvaluateEnvelope(env, decl, fingerprint, now)
	if !eval.HasKey || decl.IsCustom() || env.Provider == license.ProviderCustom || !eval.NeedsValidation || u.validator == nil {
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		return info, buildErr
	}
	u.gate.Unlock()

	val, valErr := u.validator.Validate(ctx, env.Provider, env.Key, env.InstanceID)
	if valErr != nil {
		if errors.Is(valErr, license.ErrExpired) {
			return u.applyVerifiedStatus(ctx, decl, env, license.StatusExpired, nil, false)
		}
		if errors.Is(valErr, license.ErrDisabled) || errors.Is(valErr, license.ErrNotFound) {
			return u.applyVerifiedStatus(ctx, decl, env, license.StatusDisabled, nil, false)
		}
		// Transient network or 5xx/429 failure leaves the local envelope untouched.
		u.gate.Lock()
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		return info, buildErr
	}

	return u.applyVerifiedStatus(ctx, decl, env, val.Status, val.ExpiresAt, val.Status == license.StatusActive)
}

func (u *Verify) applyVerifiedStatus(
	ctx context.Context,
	decl license.Declaration,
	env license.Envelope,
	status license.Status,
	expiresAt *time.Time,
	markVerified bool,
) (license.Info, error) {
	now := u.clock.Now().UTC()
	nowStr := license.FormatTime(now)
	prevStatus := env.Status
	env.Status = status
	if markVerified {
		env.VerifiedAt = nowStr
		env.LastSeenAt = nowStr
	}
	if expiresAt != nil {
		env.ExpiresAt = license.FormatTime(*expiresAt)
	}
	u.gate.Lock()
	_ = u.writeEnvelope(ctx, decl.ID, env)
	info, _, err := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	if prevStatus != status || markVerified {
		u.notifyChange(ctx)
	}
	return info, err
}
