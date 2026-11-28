package identity_test

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	identitylib "github.com/jiva-studio/numen/libs/shared/identity"
)

func TestAdoptChain_KeepsTheForkTheRecoveryKeySignedWhenItIsAsLongAsTheChainOfTheDevice(t *testing.T) {
	t.Parallel()
	device := newLinkDevice("Laptop", testsupport.NewClockAt(getNow()))
	device.createAccount(t)
	base := device.readChain(t)
	record := device.readRecord(t)
	rogue, err := identitylib.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	friend, err := identitylib.NewDeviceKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := identitylib.ParseRecoveryWords(getLegalWords())
	if err != nil {
		t.Fatal(err)
	}
	recovery, err := parsed.DeriveRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	byRogue, err := base.AppendDevice(record.Device, rogue.Public(), "Rogue", getNow())
	if err != nil {
		t.Fatal(err)
	}
	byRecovery, err := base.AppendDevice(recovery, friend.Public(), "Friend", getNow())
	if err != nil {
		t.Fatal(err)
	}
	if err := device.adopt().Execute(t.Context(), byRogue.Bytes()); err != nil {
		t.Fatal(err)
	}
	if byRogue.Len() != byRecovery.Len() || !byRecovery.IsPreferredOver(byRogue) {
		t.Fatal("the fixture needs a fork of equal length that the recovery key signed")
	}

	if err := device.adopt().Execute(t.Context(), byRecovery.Bytes()); err != nil {
		t.Fatalf("AdoptChain() = %v", err)
	}

	if got := device.readChain(t); !bytes.Equal(got.Bytes(), byRecovery.Bytes()) {
		t.Error("the chain file still holds the fork signed by a device, want the fork signed by the recovery key")
	}
}

func TestLinkJoiner_RefusesAGrantWhoseChainIsDatedFarAfterItsClock(t *testing.T) {
	t.Parallel()
	host := newLinkDevice("Laptop", testsupport.NewClockAt(getNow().Add(identitylib.MaxClockSkew+time.Hour)))
	joiner := newLinkDevice("Phone", testsupport.NewClockAt(getNow()))
	host.createAccount(t)
	r := newRelay(t, host)
	if _, err := host.start().Execute(t.Context(), identitydomain.LinkRoleHost, identitydomain.LinkModeShow, ""); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	if _, err := joiner.start().Execute(t.Context(), identitydomain.LinkRoleJoiner, identitydomain.LinkModeScan, host.getLink(t).QRText); err != nil {
		t.Fatal(err)
	}
	r.settle(t, host, joiner)
	approvePeer(t, r, host, joiner)

	if got := joiner.getLink(t).Phase; got != identitydomain.LinkPhaseFailed {
		t.Errorf("the joiner is in phase %v, want FAILED for a chain dated after its clock", got)
	}
}
