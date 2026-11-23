package identity_test

import (
	"encoding/base64"
	"errors"
	"testing"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func newClock() *testsupport.ManualClock { return testsupport.NewClockAt(getNow()) }

func TestCreateIdentity_KeepsTheChainInTheFileAndOnlyTheSmallRecordInTheKeyStore(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Laptop", newClock())
	device.createAccount(t)
	if got := len(device.readChain(t).ListDevices()); got != 1 {
		t.Errorf("the chain file lists %d devices, want 1", got)
	}
	raw, err := base64.RawURLEncoding.DecodeString(device.keys.snapshot()[keyName])
	if err != nil || len(raw) != 146 {
		t.Errorf("the key store record has %d bytes, error = %v, want 146", len(raw), err)
	}
}

func TestGetIdentity_ReportsAMissingChainFileAsAnError(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Laptop", newClock())
	device.createAccount(t)
	if err := device.chains.Delete(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := device.get().Execute(t.Context()); err == nil || errors.Is(err, identitydomain.ErrNoIdentity) {
		t.Errorf("error = %v, want a failure that is not ErrNoIdentity", err)
	}
}

func TestDeleteIdentity_RemovesTheRecordAndTheChainFile(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Laptop", newClock())
	device.createAccount(t)
	if err := device.remove().Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(device.keys.snapshot()) != 0 {
		t.Error("the record is still in the key store")
	}
	if _, err := device.chains.Read(t.Context()); !errors.Is(err, identitydomain.ErrNoChain) {
		t.Errorf("the chain file is still there: %v", err)
	}
}

func TestGetPublicIdentity_ReportsTheEpochTheChainLengthAndThatItIsNotRevoked(t *testing.T) {
	t.Parallel()
	l := newLinkPair(t)
	public, err := l.host.public().Execute(t.Context())
	if err != nil || public.KeyEpoch != 0 || public.ChainLength != l.host.readChain(t).Len() || public.IsRevoked {
		t.Errorf("public identity = %+v, %v", public, err)
	}
}
