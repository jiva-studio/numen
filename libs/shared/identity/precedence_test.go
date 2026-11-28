package identity_test

import (
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

type takeover struct {
	team
	attacker identity.DeviceKey
	stolen   identity.Chain
	recovery identity.Chain
	fresh    identity.AccountKeys
}

func newTakeover(t *testing.T) takeover {
	t.Helper()
	m := newTeam(t)
	attacker := newDeviceOf(t)
	chain, err := appended(m.chain, m.first, attacker.Public(), "Attacker", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	stolen, _, err := chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, rand.Reader, epoch().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	recovery, fresh, err := m.chain.RevokeDevice(m.recovery, m.keys, m.first.Public().Sign, rand.Reader, epoch().Add(3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return takeover{team: m, attacker: attacker, stolen: stolen, recovery: recovery, fresh: fresh}
}

func TestAdopt_TheEntrySignedByTheRecoveryKeyWinsAtAFork(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)
	state, err := identity.NewState(x.secondRecord(t), x.chain)
	if err != nil {
		t.Fatal(err)
	}
	state, err = state.Adopt(x.stolen)
	if err != nil || state.Status != identity.StatusRevoked {
		t.Fatalf("after the stolen device cut it out: status = %v, error = %v, want revoked", state.Status, err)
	}

	next, err := state.Adopt(x.recovery)

	if err != nil || next.Status != identity.StatusActive {
		t.Fatalf("Adopt() = %v, status %v, want the device active again", err, next.Status)
	}
	keys, err := next.OpenAccountKeys()
	if err != nil || keys.BoxPublicKey() != x.fresh.BoxPublicKey() {
		t.Errorf("the device does not hold the keys of the recovery chain: %v", err)
	}
	if next.Chain.IsActiveDevice(x.attacker.Public().Sign) || next.Chain.IsKnownDevice(x.attacker.Public().Sign) || next.Chain.IsActiveDevice(x.first.Public().Sign) {
		t.Error("the chain still lists a device of the voided entries or the stolen device")
	}
}

func TestAdopt_ADeviceEntryDoesNotReplaceTheRecoveryEntryAtAFork(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)
	state, err := identity.NewState(x.secondRecord(t), x.chain)
	if err != nil {
		t.Fatal(err)
	}
	state, err = state.Adopt(x.recovery)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := state.Adopt(x.stolen); !errors.Is(err, identity.ErrNotExtension) {
		t.Errorf("Adopt() = %v, want ErrNotExtension", err)
	}
}

func TestIsPreferredOver_LowerHashWinsBetweenRecoveryEntries(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	one, err := appended(m.chain, m.recovery, newDeviceOf(t).Public(), "Tablet", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	two, err := appended(m.chain, m.recovery, newDeviceOf(t).Public(), "Desktop", epoch().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if one.IsPreferredOver(two) == two.IsPreferredOver(one) {
		t.Error("exactly one of two recovery forks must outrank the other")
	}
}

func TestAdopt_ForkWithoutDeviceRejected(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)
	chain, err := appended(x.chain, x.first, x.attacker.Public(), "Attacker", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	wrapped, err := x.keys.WrapForDevice(x.attacker.Public())
	if err != nil {
		t.Fatal(err)
	}
	state, err := identity.NewState(identity.Record{Device: x.attacker, Wrapped: wrapped}, chain)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := state.Adopt(x.recovery); !errors.Is(err, identity.ErrDeviceVoided) {
		t.Errorf("Adopt() = %v, want ErrDeviceVoided", err)
	}
}

func TestListDeviceRevocations_ListsOnlyTheRevokesSignedByADevice(t *testing.T) {
	t.Parallel()
	x := newTakeover(t)

	byDevice := x.stolen.ListDeviceRevocations()
	byWords := x.recovery.ListDeviceRevocations()

	if len(byDevice) != 1 || string(byDevice[0].By) != string(x.first.Public().Sign) || string(byDevice[0].Device) != string(x.second.Public().Sign) {
		t.Errorf("revocations = %+v, want the revoke of the second device by the first", byDevice)
	}
	if !byDevice[0].At.Equal(epoch().Add(2*time.Hour)) || len(byWords) != 0 {
		t.Errorf("at = %v, by words = %+v, want the time of the entry and no revoke by the words", byDevice[0].At, byWords)
	}
}
