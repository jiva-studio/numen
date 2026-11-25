package license_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/license"
	licenseusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/license"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func syncPlugins() []manifest.DiscoveredPlugin {
	return []manifest.DiscoveredPlugin{{
		Manifest: manifest.Manifest{
			ID:   "md.numen.tools.sync",
			Name: "Numen Sync",
			Contributes: manifest.Contributes{
				Licenses: []manifest.License{{
					ID:          "sync",
					Title:       "Numen Sync",
					Description: "Synchronizes your vaults across devices",
					Providers: []manifest.LicenseProvider{
						{Provider: "creem", ProductID: "prod_sync"},
						{Provider: "dodo", ProductID: "pdt_sync"},
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
		activateResult: license.ActivationResult{
			Provider:   "creem",
			InstanceID: "li_101",
			Status:     license.StatusActive,
			ExpiresAt:  &expires,
		},
		validateResult: license.ValidationResult{
			Status:    license.StatusActive,
			ExpiresAt: &expires,
		},
	}
	clock := &fakeClock{now: now}
	changes := newFakeChangeHub()
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: syncPlugins()},
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
		LicenseID: "sync",
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
		LicenseID: "sync",
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
	if _, err := deactivateUC.Execute(t.Context(), "sync"); !errors.Is(err, license.ErrProviderUnavailable) {
		t.Fatalf("Deactivate(offline) error = %v, want ErrProviderUnavailable", err)
	}
	afterFailedDeactivate, err := listUC.Execute(t.Context())
	if err != nil || !afterFailedDeactivate[0].HasKey || !afterFailedDeactivate[0].IsActive {
		t.Fatalf("license after failed deactivate = %+v, err = %v", afterFailedDeactivate, err)
	}

	// Deactivate succeeds when provider returns 200 OK or 404 ErrNotFound.
	validator.deactivateErr = license.ErrNotFound
	deactivated, err := deactivateUC.Execute(t.Context(), "sync")
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
		activateResult: license.ActivationResult{
			Provider:   "dodo",
			InstanceID: "lki_55",
			Status:     license.StatusActive,
		},
		validateResult: license.ValidationResult{
			Status: license.StatusExpired,
		},
	}
	clock := &fakeClock{now: now}
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: syncPlugins()},
		keys,
		validator,
		fakeMachine{fingerprint: "fp-local", instanceName: "akd-laptop"},
		clock,
		newFakeChangeHub(),
	)

	activateUC := licenseusecase.NewActivate(deps)
	verifyUC := licenseusecase.NewVerify(deps)

	if _, err := activateUC.Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "sync",
		Key:       "DODO-1234-9999",
	}); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	// Advance clock by 2 hours (past 1h FreshnessTTL) and verify.
	clock.set(now.Add(2 * time.Hour))
	verified, err := verifyUC.Execute(t.Context(), "sync")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if verified.IsActive || verified.Status != license.StatusExpired || !verified.HasKey {
		t.Fatalf("verified = %+v, want expired with HasKey=true", verified)
	}
}

func TestVerifyDoesNotRestoreDeactivatedLicense(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	keys := newFakeKeyStore()
	validator := &fakeValidator{
		activateResult: license.ActivationResult{
			Provider:   "creem",
			InstanceID: "li_101",
			Status:     license.StatusActive,
		},
		validateResult: license.ValidationResult{Status: license.StatusActive},
	}
	clock := &fakeClock{now: now}
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: syncPlugins()},
		keys,
		validator,
		fakeMachine{fingerprint: "fp-local", instanceName: "akd-laptop"},
		clock,
		newFakeChangeHub(),
	)
	activateUC := licenseusecase.NewActivate(deps)
	verifyUC := licenseusecase.NewVerify(deps)
	deactivateUC := licenseusecase.NewDeactivate(deps)
	listUC := licenseusecase.NewList(deps)

	if _, err := activateUC.Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "sync",
		Key:       "CREEM-ABCD-814A",
	}); err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	clock.set(now.Add(2 * time.Hour))
	validator.validateStarted = make(chan struct{})
	validator.validateRelease = make(chan struct{})
	verifyDone := make(chan error, 1)
	go func() {
		_, err := verifyUC.Execute(t.Context(), "sync")
		verifyDone <- err
	}()
	<-validator.validateStarted

	if _, err := deactivateUC.Execute(t.Context(), "sync"); err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	close(validator.validateRelease)
	if err := <-verifyDone; err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	licenses, err := listUC.Execute(t.Context())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if licenses[0].HasKey || licenses[0].IsActive {
		t.Fatalf("license = %+v, want deactivated license", licenses[0])
	}
}

func TestActivateRejectsNonActiveProviderStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	validator := &fakeValidator{activateResult: license.ActivationResult{
		Provider: "creem", InstanceID: "li_101", Status: license.StatusExpired,
	}}
	deps := licenseusecase.NewDeps(
		&fakeDeclarations{plugins: syncPlugins()}, newFakeKeyStore(), validator,
		fakeMachine{fingerprint: "fp-local", instanceName: "akd-laptop"},
		&fakeClock{now: now}, newFakeChangeHub(),
	)
	_, err := licenseusecase.NewActivate(deps).Execute(t.Context(), licenseusecase.ActivateRequest{
		LicenseID: "sync", Key: "CREEM-ABCD-814A",
	})
	if !errors.Is(err, license.ErrExpired) {
		t.Fatalf("Activate() error = %v, want ErrExpired", err)
	}
}
