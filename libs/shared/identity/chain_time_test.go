package identity_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func TestChainVerifyAt_RejectsAnEntryFarAfterTheClockOfTheAdopter(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	chain, err := a.chain.AppendDevice(a.recovery, device.Public(), "Phone", epoch().Add(identity.MaxClockSkew+time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if err := chain.VerifyAt(epoch()); !errors.Is(err, identity.ErrEntryTime) {
		t.Errorf("VerifyAt() error = %v, want ErrEntryTime", err)
	}
	if err := chain.VerifyAt(epoch().Add(identity.MaxClockSkew + time.Hour)); err != nil {
		t.Errorf("VerifyAt() at the time of the entry error = %v, want none", err)
	}
}

func TestChainVerifyAt_AcceptsAnEntryWithinTheSkew(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	device := newDevice(t)
	chain, err := a.chain.AppendDevice(a.recovery, device.Public(), "Phone", epoch().Add(identity.MaxClockSkew-time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if err := chain.VerifyAt(epoch()); err != nil {
		t.Errorf("VerifyAt() error = %v, want none", err)
	}
}

func TestListDeviceAdditions_ListsOnlyAdditionsSignedByADevice(t *testing.T) {
	t.Parallel()
	a := newChain(t)
	first, second := newDevice(t), newDevice(t)
	chain, err := a.chain.AppendDevice(a.recovery, first.Public(), "Laptop", epoch())
	if err != nil {
		t.Fatal(err)
	}
	chain, err = chain.AppendDevice(first, second.Public(), "Phone", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	got := chain.ListDeviceAdditions()

	if len(got) != 1 || string(got[0].By) != string(first.Public().Sign) || string(got[0].Device) != string(second.Public().Sign) || !got[0].At.Equal(epoch().Add(time.Hour)) {
		t.Errorf("additions = %+v, want the one the Laptop signed", got)
	}
}
