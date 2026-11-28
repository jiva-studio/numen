package identity_test

import (
	"crypto/rand"
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func newAccount(t *testing.T) identity.AccountKeys {
	t.Helper()
	account, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return account
}

func newDevice(t *testing.T) identity.DeviceKey {
	t.Helper()
	device, err := identity.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return device
}

func TestWrapForRecovery_IsOpenedByTheRecoveryKeyOfTheWords(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	recovery := deriveRecovery(t, abandonWords(t))
	wrapped, err := account.WrapForRecovery(recovery.Public())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(wrapped.Bytes()); got != 112 {
		t.Errorf("wrapped keys have %d bytes, want 112", got)
	}
	opened, err := recovery.UnwrapAccountKeys(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(opened, account) {
		t.Error("the unwrapped keys differ from the account keys")
	}
	if opened.BoxPublicKey() != account.BoxPublicKey() {
		t.Error("the unwrapped box key differs")
	}
}

func TestWrapForDevice_IsOpenedByThatDeviceOnly(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	device, other := newDevice(t), newDevice(t)
	wrapped, err := account.WrapForDevice(device.Public())
	if err != nil {
		t.Fatal(err)
	}
	opened, err := device.UnwrapAccountKeys(wrapped)
	if err != nil || !reflect.DeepEqual(opened, account) {
		t.Fatalf("device unwrap = %v, %v", opened, err)
	}
	if _, err := other.UnwrapAccountKeys(wrapped); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("another device unwrap error = %v, want ErrUnwrap", err)
	}
}

func TestUnwrapAccountKeys_RefusesAnotherKey(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	wrapped := wrapForRecovery(t, account, deriveRecovery(t, abandonWords(t)).Public())
	other := parseWords(t, bip39Vectors()[1].words)
	if _, err := deriveRecovery(t, other).UnwrapAccountKeys(wrapped); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("error = %v, want ErrUnwrap", err)
	}
	if _, err := newDevice(t).UnwrapAccountKeys(wrapped); !errors.Is(err, identity.ErrUnwrap) {
		t.Errorf("a device opened keys wrapped for the recovery key: %v", err)
	}
}

func TestUnwrapAccountKeys_RefusesAnyChangedByte(t *testing.T) {
	t.Parallel()
	recovery := deriveRecovery(t, abandonWords(t))
	wrapped := wrapForRecovery(t, newAccount(t), recovery.Public())
	sealed := wrapped.Bytes()
	for i := range sealed {
		changed := append([]byte(nil), sealed...)
		changed[i] ^= 0x01
		parsed, err := identity.ParseWrappedAccountKeys(changed)
		if err != nil {
			t.Fatalf("byte %d: ParseWrappedAccountKeys error = %v", i, err)
		}
		if _, err := recovery.UnwrapAccountKeys(parsed); !errors.Is(err, identity.ErrUnwrap) {
			t.Fatalf("byte %d changed: error = %v, want ErrUnwrap", i, err)
		}
	}
}

func TestWrappedAccountKeys_ParseOnlyTheRightLength(t *testing.T) {
	t.Parallel()
	wrapped := wrapForRecovery(t, newAccount(t), deriveRecovery(t, abandonWords(t)).Public())
	parsed, err := identity.ParseWrappedAccountKeys(wrapped.Bytes())
	if err != nil || string(parsed.Bytes()) != string(wrapped.Bytes()) {
		t.Fatalf("round trip = %v", err)
	}
	for _, size := range []int{0, 111, 113} {
		if _, err := identity.ParseWrappedAccountKeys(make([]byte, size)); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("%d bytes: error = %v, want ErrMalformed", size, err)
		}
	}
}

func TestWrapForRecovery_ReturnsADifferentCiphertextEachTime(t *testing.T) {
	t.Parallel()
	account := newAccount(t)
	public := deriveRecovery(t, abandonWords(t)).Public()
	first := wrapForRecovery(t, account, public)
	second := wrapForRecovery(t, account, public)
	if string(first.Bytes()) == string(second.Bytes()) {
		t.Error("two wraps of the same keys are equal")
	}
}

func TestNewAccountKeys_AreNewEachTime(t *testing.T) {
	t.Parallel()
	first, second := newAccount(t).BoxPublicKey(), newAccount(t).BoxPublicKey()
	if first == second {
		t.Error("two account keys are equal")
	}
}
