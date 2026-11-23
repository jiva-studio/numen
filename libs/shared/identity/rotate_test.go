package identity_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestRotateKeys_StartsTheNextEpochForEveryActiveDeviceAndRemovesNone(t *testing.T) {
	t.Parallel()
	m := newTeam(t)

	chain, fresh, err := m.chain.RotateKeys(m.recovery, m.keys, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		t.Fatalf("RotateKeys() = %v", err)
	}

	if err := chain.Verify(); err != nil {
		t.Fatalf("Verify() = %v", err)
	}
	if chain.Epoch() != m.chain.Epoch()+1 || fresh.BoxPublicKey() != chain.AccountBoxKey() || fresh.BoxPublicKey() == m.keys.BoxPublicKey() {
		t.Errorf("epoch = %d, box = %x, want the next epoch with new keys", chain.Epoch(), chain.AccountBoxKey())
	}
	if len(chain.ListDevices()) != 2 || len(chain.ListRevokedDevices()) != 0 {
		t.Errorf("active = %d, revoked = %d, want 2 and 0", len(chain.ListDevices()), len(chain.ListRevokedDevices()))
	}
	for _, device := range []identity.DeviceKey{m.first, m.second} {
		wrapped, ok := chain.GetWrappedFor(device.Public().Sign)
		if !ok {
			t.Fatal("a device has no keys of the new epoch")
		}
		opened, err := device.UnwrapAccountKeys(wrapped)
		if err != nil || opened.BoxPublicKey() != fresh.BoxPublicKey() {
			t.Errorf("a device opened %v with error %v, want the new keys", opened.BoxPublicKey(), err)
		}
	}
	if recovered, err := m.recovery.UnwrapAccountKeys(chain.WrappedForRecovery()); err != nil || recovered.BoxPublicKey() != fresh.BoxPublicKey() {
		t.Errorf("the recovery key opened %v with error %v, want the new keys", recovered.BoxPublicKey(), err)
	}
}

func TestChainMark_IsContainedByTheChainItWasTakenFromAndByItsExtensionsOnly(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	mark := m.chain.GetMark()
	longer, _, err := m.chain.RotateKeys(m.recovery, m.keys, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	other := newTeam(t)
	shorter, err := identity.ParseChain(m.chain.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if !m.chain.Contains(mark) || !longer.Contains(mark) {
		t.Error("the chain or its extension does not contain its own mark")
	}
	if other.chain.Contains(mark) {
		t.Error("another history contains the mark")
	}
	if mark.Length != shorter.Len() || shorter.Len() >= longer.Len() {
		t.Errorf("mark length = %d, chain = %d, longer = %d", mark.Length, shorter.Len(), longer.Len())
	}
	parsed, err := identity.ParseMark(" " + mark.String() + " ")
	if err != nil || parsed != mark {
		t.Errorf("ParseMark(%q) = %v, %v", mark.String(), parsed, err)
	}
	if _, err := identity.ParseMark("7-xyz"); !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("ParseMark of a bad text = %v, want ErrMalformed", err)
	}
}
