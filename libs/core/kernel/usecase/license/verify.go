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
	info, ok := res.(license.Info)
	if !ok {
		return license.Info{}, nil
	}
	return info, nil
}

func (u *Verify) verifyOnce(ctx context.Context, licenseID string) (license.Info, error) {
	decl, err := u.findDeclaration(licenseID)
	if err != nil {
		return license.Info{}, err
	}
	env, shouldVerify, info, err := u.prepareVerification(ctx, decl)
	if err != nil || !shouldVerify {
		return info, err
	}
	val, valErr := u.validator.Validate(ctx, env.Provider, env.Key, env.InstanceID)
	if valErr != nil {
		return u.handleValidationFailure(ctx, decl, env, valErr)
	}
	return u.applyVerifiedStatus(ctx, decl, env, val.Status, val.ExpiresAt, val.Status == license.StatusActive)
}

func (u *Verify) prepareVerification(
	ctx context.Context,
	decl license.Declaration,
) (license.Envelope, bool, license.Info, error) {
	u.gate.Lock()
	defer u.gate.Unlock()
	env, isFound, err := u.readEnvelope(ctx, decl.ID)
	if err != nil {
		return license.Envelope{}, false, license.Info{}, err
	}
	if !isFound {
		info, _, buildErr := u.buildInfo(ctx, decl)
		return license.Envelope{}, false, info, buildErr
	}
	fingerprint := ""
	if u.machine != nil {
		fingerprint = u.machine.GetMachineFingerprint()
	}
	eval := license.EvaluateEnvelope(env, decl, fingerprint, u.clock.Now().UTC())
	if !canVerifyOnline(decl, env, eval) || u.validator == nil {
		info, _, buildErr := u.buildInfo(ctx, decl)
		return license.Envelope{}, false, info, buildErr
	}
	return env, true, license.Info{}, nil
}

func canVerifyOnline(decl license.Declaration, env license.Envelope, eval license.Evaluation) bool {
	return eval.HasKey && !decl.IsCustom() && env.Provider != license.ProviderCustom && eval.NeedsValidation
}

func (u *Verify) handleValidationFailure(
	ctx context.Context,
	decl license.Declaration,
	env license.Envelope,
	valErr error,
) (license.Info, error) {
	if errors.Is(valErr, license.ErrExpired) {
		return u.applyVerifiedStatus(ctx, decl, env, license.StatusExpired, nil, false)
	}
	if errors.Is(valErr, license.ErrDisabled) || errors.Is(valErr, license.ErrNotFound) {
		return u.applyVerifiedStatus(ctx, decl, env, license.StatusDisabled, nil, false)
	}
	u.gate.Lock()
	info, _, buildErr := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	return info, buildErr
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
	current, isFound, readErr := u.readEnvelope(ctx, decl.ID)
	if readErr != nil {
		u.gate.Unlock()
		return license.Info{}, readErr
	}
	if !isFound || !isSameActivation(current, env) {
		info, _, buildErr := u.buildInfo(ctx, decl)
		u.gate.Unlock()
		return info, buildErr
	}
	if writeErr := u.writeEnvelope(ctx, decl.ID, env); writeErr != nil {
		u.gate.Unlock()
		return license.Info{}, writeErr
	}
	info, _, err := u.buildInfo(ctx, decl)
	u.gate.Unlock()
	if prevStatus != status || markVerified {
		u.notifyChange(ctx)
	}
	return info, err
}

func isSameActivation(current, expected license.Envelope) bool {
	return license.NormalizeKey(current.Key) == license.NormalizeKey(expected.Key) &&
		current.Provider == expected.Provider &&
		current.InstanceID == expected.InstanceID &&
		current.MachineFingerprint == expected.MachineFingerprint
}
