package identity_test

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestRevokeOthers_KeepsOnlyTheNamedDeviceAndMovesTheKeysToTheNextEpoch(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	restored := newDeviceOf(t)
	withNew, err := appended(m.chain, m.recovery, restored.Public(), "Restored", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	chain, fresh, err := withNew.RevokeOthers(m.recovery, m.keys, restored.Public().Sign, rand.Reader, epoch().Add(2*time.Hour))
	if err != nil {
		t.Fatalf("RevokeOthers() = %v", err)
	}

	if err := chain.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if active := chain.ListDevices(); len(active) != 1 || !bytes.Equal(active[0].Public.Sign, restored.Public().Sign) {
		t.Errorf("active devices = %+v, want only the restored one", active)
	}
	if revoked := chain.ListRevokedDevices(); len(revoked) != 2 {
		t.Errorf("revoked devices = %d, want the two old ones", len(revoked))
	}
	for _, old := range []identity.DeviceKey{m.first, m.second} {
		if _, ok := chain.GetWrappedFor(old.Public().Sign); ok {
			t.Error("an old device has keys of the new epoch")
		}
	}
	wrapped, ok := chain.GetWrappedFor(restored.Public().Sign)
	if !ok {
		t.Fatal("the restored device has no keys of the new epoch")
	}
	if opened, err := restored.UnwrapAccountKeys(wrapped); err != nil || opened.BoxPublicKey() != fresh.BoxPublicKey() {
		t.Errorf("the restored device opened %v with error %v, want the new keys", opened.BoxPublicKey(), err)
	}
	if got := chain.GetRevokedSince(withNew); len(got) != 2 {
		t.Errorf("GetRevokedSince() = %d keys, want the two old devices", len(got))
	}
}

func TestRevokeOthers_RefusesADeviceThatTheChainDoesNotList(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	if _, _, err := m.chain.RevokeOthers(m.recovery, m.keys, newDeviceOf(t).Public().Sign, rand.Reader, epoch().Add(time.Hour)); err == nil {
		t.Error("RevokeOthers() kept a device that is not in the chain")
	}
}
