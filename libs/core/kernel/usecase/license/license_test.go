package license_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	"github.com/jiva-studio/numen/libs/core/kernel/port/licenseport"
	licenseusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func supporterPlugins() []manifest.Discovered {
	return []manifest.Discovered{{
		Manifest: manifest.Manifest{
			ID:   "md.numen.tools.supporter",
			Name: "Supporter",
			Contributes: manifest.Contributes{
				Licenses: []manifest.License{{
					ID:          "supporter",
					Title:       "Supporter",
					Description: "Shows the Supporter badge in the title bar",
					Providers: []manifest.LicenseProvider{
						{Provider: "creem", ProductID: "prod_supporter"},
						{Provider: "dodo", ProductID: "pdt_supporter"},
					},
				}},
			},
		},
	}}
}

func TestActivateIdempotencyAndDeactivateConfirmation(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	expires := now.Add(30 * 24 * time.Hour)
	keys := newFakeKeyStore()
	validator := &fakeValidator{
		activateResult: licenseport.ActivationResult{
			Provider:   "creem",
			InstanceID: "li_101",
			Status:     license.StatusActive,
			ExpiresAt:  &expires,
		},
		validateResult: licenseport.ValidationResult{
			Status:    license.StatusActive,
			ExpiresAt: &expires,
		},
	}
	clock := &fakeClock{now: now}
	changes := newFakeChangeHub()
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: supporterPlugins()},
		keys,
		validator,
		fakeMachine{fingerprint: "fp-local", instanceName: "akd-laptop"},
		clock,
		changes,
	)

	listUC := licenseusecase.NewList(deps)
	activateUC := licenseusecase.NewActivate(deps)
	deactivateUC := licenseusecase.NewDeactivate(deps)

	// Initial list shows inactive license with 2 provider options.
	initial, err := listUC.Execute(t.Context())
	if err != nil || len(initial) != 1 {
		t.Fatalf("List() = %+v, err = %v", initial, err)
	}
	if initial[0].IsActive || initial[0].HasKey || len(initial[0].Providers) != 2 {
		t.Fatalf("initial license = %+v", initial[0])
	}

	// Activate key.
	activated, err := activateUC.Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "supporter",
		Key:       "  CREEM-ABCD-814A \n",
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if !activated.IsActive || !activated.HasKey || activated.KeyHint != "••••-814A" || activated.ActiveProvider != "creem" {
		t.Fatalf("activated = %+v", activated)
	}
	if validator.activateCalls != 1 {
		t.Fatalf("activateCalls = %d, want 1", validator.activateCalls)
	}

	// Re-activating the same key on the same machine calls Validate rather than burning another slot.
	reActivated, err := activateUC.Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "supporter",
		Key:       "CREEM-ABCD-814A",
	})
	if err != nil || !reActivated.IsActive {
		t.Fatalf("re-Activate() = %+v, err = %v", reActivated, err)
	}
	if validator.activateCalls != 1 || validator.validateCalls != 1 {
		t.Fatalf("activateCalls=%d validateCalls=%d, want 1 and 1", validator.activateCalls, validator.validateCalls)
	}

	// Deactivate fails when provider is unreachable and preserves the key in KeyStore.
	validator.deactivateErr = license.ErrProviderUnavailable
	if _, err := deactivateUC.Execute(t.Context(), "supporter"); !errors.Is(err, license.ErrProviderUnavailable) {
		t.Fatalf("Deactivate(offline) error = %v, want ErrProviderUnavailable", err)
	}
	afterFailedDeactivate, err := listUC.Execute(t.Context())
	if err != nil || !afterFailedDeactivate[0].HasKey || !afterFailedDeactivate[0].IsActive {
		t.Fatalf("license after failed deactivate = %+v, err = %v", afterFailedDeactivate, err)
	}

	// Deactivate succeeds when provider returns 200 OK or 404 ErrNotFound.
	validator.deactivateErr = license.ErrNotFound
	deactivated, err := deactivateUC.Execute(t.Context(), "supporter")
	if err != nil {
		t.Fatalf("Deactivate(404) error = %v", err)
	}
	if deactivated.HasKey || deactivated.IsActive || deactivated.Status != license.StatusInactive {
		t.Fatalf("deactivated = %+v, want inactive with HasKey=false", deactivated)
	}
}

func TestVerifyUpdatesStatusAfterTTL(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	keys := newFakeKeyStore()
	validator := &fakeValidator{
		activateResult: licenseport.ActivationResult{
			Provider:   "dodo",
			InstanceID: "lki_55",
			Status:     license.StatusActive,
		},
		validateResult: licenseport.ValidationResult{
			Status: license.StatusExpired,
		},
	}
	clock := &fakeClock{now: now}
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: supporterPlugins()},
		keys,
		validator,
		fakeMachine{fingerprint: "fp-local", instanceName: "akd-laptop"},
		clock,
		newFakeChangeHub(),
	)

	activateUC := licenseusecase.NewActivate(deps)
	verifyUC := licenseusecase.NewVerify(deps)

	if _, err := activateUC.Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "supporter",
		Key:       "DODO-1234-9999",
	}); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	// Advance clock by 2 hours (past 1h FreshnessTTL) and verify.
	clock.set(now.Add(2 * time.Hour))
	verified, err := verifyUC.Execute(t.Context(), "supporter")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.IsActive || verified.Status != license.StatusExpired || !verified.HasKey {
		t.Fatalf("verified = %+v, want expired with HasKey=true", verified)
	}
}
