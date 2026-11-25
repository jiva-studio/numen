package identity_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

// fullChain is a chain that a device key has filled up to the limit of device entries.
type fullChain struct {
	recovery identity.RecoveryKey
	chain    identity.Chain
	keys     identity.AccountKeys
	device   identity.DeviceKey
}

func fillChain(tb testing.TB) (fullChain, error) {
	tb.Helper()
	a := newChainOf(tb)
	signer := newDeviceOf(tb)
	chain, err := appended(a.chain, a.recovery, signer.Public(), "Signer", epoch())
	if err != nil {
		tb.Fatal(err)
	}
	keys := a.keys
	for i := range 2000 {
		at := epoch().Add(time.Duration(i+1) * time.Second)
		next, err := appended(chain, signer, newDeviceOf(tb).Public(), "Guest", at)
		if err != nil {
			return fullChain{recovery: a.recovery, chain: chain, keys: keys, device: signer}, err
		}
		guest := next.ListDevices()[len(next.ListDevices())-1].Public.Sign
		revoked, nextKeys, err := next.RevokeDevice(signer, keys, guest, rand.Reader, at)
		if err != nil {
			return fullChain{recovery: a.recovery, chain: chain, keys: keys, device: signer}, err
		}
		chain, keys = revoked, nextKeys
	}
	tb.Fatal("2000 devices did not reach the limit of the chain")
	return fullChain{}, nil
}

func TestChain_RefusesADeviceEntryThatMakesItLongerThanTheDeviceLimit(t *testing.T) {
	t.Parallel()
	full, err := fillChain(t)
	if !errors.Is(err, identity.ErrChainTooLong) {
		t.Fatalf("error = %v, want ErrChainTooLong", err)
	}
	limit := identity.MaxChainBytes - identity.RecoveryReserveBytes
	if size := len(full.chain.Bytes()); size > limit || size < limit-2048 {
		t.Errorf("the last chain that fits has %d bytes, want just under %d", size, limit)
	}
	if _, err := identity.ParseChain(full.chain.Bytes()); err != nil {
		t.Errorf("a chain at the limit does not parse: %v", err)
	}
}

func TestChain_RecoveryKeyRemovesTheDeviceThatFilledTheChain(t *testing.T) {
	t.Parallel()
	full, err := fillChain(t)
	if !errors.Is(err, identity.ErrChainTooLong) {
		t.Fatalf("error = %v, want ErrChainTooLong", err)
	}
	at := epoch().Add(24 * time.Hour)
	revoked, keys, err := full.chain.RevokeDevice(full.recovery, full.keys, full.device.Public().Sign, rand.Reader, at)
	if err != nil {
		t.Fatalf("a revoke signed by the recovery key does not fit: %v", err)
	}
	if revoked.IsActiveDevice(full.device.Public().Sign) {
		t.Error("the device that filled the chain is still active")
	}
	if _, _, err := revoked.RotateKeys(full.recovery, keys, rand.Reader, at.Add(time.Hour)); err != nil {
		t.Errorf("a rotation signed by the recovery key does not fit after the revoke: %v", err)
	}
	if _, err := identity.ParseChain(revoked.Bytes()); err != nil {
		t.Errorf("the chain with the revoke does not parse: %v", err)
	}
}

func TestChain_RefusesADeviceEntryAfterTheRecoveryKeyUsedTheReserve(t *testing.T) {
	t.Parallel()
	full, err := fillChain(t)
	if !errors.Is(err, identity.ErrChainTooLong) {
		t.Fatalf("error = %v, want ErrChainTooLong", err)
	}
	at := epoch().Add(24 * time.Hour)
	withRecovery, err := appended(full.chain, full.recovery, newDeviceOf(t).Public(), "Owner", at)
	if err != nil {
		t.Fatalf("an entry signed by the recovery key does not fit: %v", err)
	}
	if _, err := appended(withRecovery, full.device, newDeviceOf(t).Public(), "Guest", at.Add(time.Hour)); !errors.Is(err, identity.ErrChainTooLong) {
		t.Errorf("a device entry after the recovery entry: error = %v, want ErrChainTooLong", err)
	}
}

func TestParseChain_RefusesMoreThanTheHardLimitBeforeReadingIt(t *testing.T) {
	t.Parallel()
	full, err := fillChain(t)
	if !errors.Is(err, identity.ErrChainTooLong) {
		t.Fatal(err)
	}
	oversized := append(full.chain.Bytes(), make([]byte, identity.MaxChainBytes)...)
	if _, err := identity.ParseChain(oversized); !errors.Is(err, identity.ErrChainTooLong) {
		t.Errorf("error = %v, want ErrChainTooLong", err)
	}
}
