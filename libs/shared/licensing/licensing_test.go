package licensing_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/licensing"
)

func TestNormalizeKeyStripsSpacesAndInvisibleCharacters(t *testing.T) {
	t.Parallel()
	got := licensing.NormalizeKey(" \t\n\u200bCREEM-1234-ABCD\u200b \r\n")
	if got != "CREEM-1234-ABCD" {
		t.Fatalf("NormalizeKey() = %q, want %q", got, "CREEM-1234-ABCD")
	}
}

func TestGetKeyHintMasksAllButLastFourRunes(t *testing.T) {
	t.Parallel()
	if got := licensing.GetKeyHint(""); got != "" {
		t.Fatalf("GetKeyHint(\"\") = %q, want empty", got)
	}
	if got := licensing.GetKeyHint("ABC"); got != "••••" {
		t.Fatalf("GetKeyHint(\"ABC\") = %q, want \"••••\"", got)
	}
	if got := licensing.GetKeyHint("  CREEM-1234-814A\n"); got != "••••-814A" {
		t.Fatalf("GetKeyHint() = %q, want \"••••-814A\"", got)
	}
}

func TestSelectErrorPicksMostSpecificFailure(t *testing.T) {
	t.Parallel()
	notFound := fmt.Errorf("creem: %w", licensing.ErrLicenseNotFound)
	unavailable := fmt.Errorf("creem: %w", licensing.ErrProviderUnavailable)
	mismatch := fmt.Errorf("dodo: %w", licensing.ErrProductMismatch)
	disabled := fmt.Errorf("dodo: %w", licensing.ErrLicenseDisabled)
	expired := fmt.Errorf("creem: %w", licensing.ErrLicenseExpired)
	limitReached := fmt.Errorf("dodo: %w", licensing.ErrActivationLimitReached)

	if got := licensing.SelectError([]error{notFound, limitReached}); !errors.Is(got, licensing.ErrActivationLimitReached) {
		t.Fatalf("SelectError([notFound, limitReached]) = %v, want ErrActivationLimitReached", got)
	}
	if got := licensing.SelectError([]error{notFound, expired, disabled}); !errors.Is(got, licensing.ErrLicenseExpired) {
		t.Fatalf("SelectError([notFound, expired, disabled]) = %v, want ErrLicenseExpired", got)
	}
	if got := licensing.SelectError([]error{unavailable, mismatch, notFound}); !errors.Is(got, licensing.ErrProductMismatch) {
		t.Fatalf("SelectError([unavailable, mismatch, notFound]) = %v, want ErrProductMismatch", got)
	}
	if got := licensing.SelectError([]error{nil, notFound}); !errors.Is(got, licensing.ErrLicenseNotFound) {
		t.Fatalf("SelectError([nil, notFound]) = %v, want ErrLicenseNotFound", got)
	}
	if got := licensing.SelectError(nil); got != nil {
		t.Fatalf("SelectError(nil) = %v, want nil", got)
	}
}

func TestGetProviderInfoReportsKnownProviders(t *testing.T) {
	t.Parallel()
	if !licensing.IsKnownProvider(licensing.ProviderIDCreem) {
		t.Fatal("expected creem to be a known provider")
	}
	if !licensing.IsKnownProvider(licensing.ProviderIDDodo) {
		t.Fatal("expected dodo to be a known provider")
	}
	if licensing.IsKnownProvider("unknown") {
		t.Fatal("expected unknown to not be a known provider")
	}
	if !licensing.StatusActive.IsActive() {
		t.Fatal("expected StatusActive.IsActive() to be true")
	}
	if licensing.StatusExpired.IsActive() {
		t.Fatal("expected StatusExpired.IsActive() to be false")
	}
}
