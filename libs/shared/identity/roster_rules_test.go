package identity_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
)

func (m team) rotate(tb testing.TB, at time.Time) (identity.Chain, identity.AccountKeys) {
	tb.Helper()
	chain, keys, err := m.chain.RotateKeys(m.recovery, m.keys, rand.Reader, at)
	if err != nil {
		tb.Fatalf("RotateKeys() = %v", err)
	}
	return chain, keys
}

func TestRevokeDevice_KeepsTheReasonAndTheDateOfAStolenDevice(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	since := epoch().Add(30 * time.Minute)
	chain, _, err := m.chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, identity.Revocation{Reason: identity.ReasonStolen, Since: since}, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		t.Fatalf("RevokeDevice() = %v", err)
	}
	parsed, err := identity.ParseChain(chain.Bytes())
	if err != nil {
		t.Fatalf("ParseChain() = %v", err)
	}
	removal := parsed.ListRevokedDevices()[0].Removal
	if removal.Reason != identity.ReasonStolen || !removal.Since.Equal(since) || !removal.At.Equal(epoch().Add(time.Hour)) || !bytes.Equal(removal.By, m.first.Public().Sign) {
		t.Errorf("removal = %+v, want stolen since %v, at the revoke, by the laptop", removal, since)
	}
}

func TestRevokeDevice_RefusesAStolenDeviceWithoutADate(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, _, err := m.chain.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, identity.Revocation{Reason: identity.ReasonStolen}, rand.Reader, epoch().Add(time.Hour))
	if !errors.Is(err, identity.ErrMalformed) {
		t.Errorf("error = %v, want ErrMalformed", err)
	}
}

func TestRotateKeys_StartsANewEpochForEveryActiveDeviceAndRemovesNone(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, keys := m.rotate(t, epoch().Add(time.Hour))
	if len(chain.ListDevices()) != 2 || len(chain.ListRevokedDevices()) != 0 {
		t.Fatalf("active = %d, revoked = %d, want both devices active", len(chain.ListDevices()), len(chain.ListRevokedDevices()))
	}
	if chain.Epoch() != 1 || keys.Epoch() != 1 || chain.AccountBoxKey() != keys.BoxPublicKey() {
		t.Errorf("epoch = %d, keys epoch = %d, want 1 and the box of the new keys", chain.Epoch(), keys.Epoch())
	}
	for _, device := range []identity.DeviceKey{m.first, m.second} {
		wrapped, ok := chain.GetWrappedFor(device.Public().Sign)
		if !ok {
			t.Fatal("a device has no new keys")
		}
		if opened, err := device.UnwrapAccountKeys(wrapped); err != nil || opened.BoxPublicKey() != keys.BoxPublicKey() {
			t.Errorf("a device cannot open the new keys: %v", err)
		}
	}
}

func TestRotateKeys_LetsTheRecoveryKeyOpenTheHistory(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	oldSealed, err := m.firstKeys.SealForPlugin(rand.Reader, sealPluginID, []byte(revokeContext), []byte("eggs"))
	if err != nil {
		t.Fatal(err)
	}
	chain, _ := m.rotate(t, epoch().Add(time.Hour))
	latest, err := m.recovery.UnwrapAccountKeys(chain.WrappedForRecovery())
	if err != nil {
		t.Fatal(err)
	}
	history, err := chain.OpenKeyHistory(latest)
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := history.OpenForPlugin(sealPluginID, []byte(revokeContext), oldSealed); err != nil || string(opened) != "eggs" {
		t.Errorf("epoch 0 opened = %q, %v, want eggs", opened, err)
	}
}

func TestRotateKeys_RoundTripsThroughBytes(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.rotate(t, epoch().Add(time.Hour))
	parsed, err := identity.ParseChain(chain.Bytes())
	if err != nil || !bytes.Equal(parsed.Bytes(), chain.Bytes()) || len(parsed.ListDevices()) != 2 || parsed.Epoch() != 1 {
		t.Errorf("parsed chain differs: %v", err)
	}
}

func TestRotateKeys_LetsAnActiveDeviceTakeItsNewKeysAndAFutureRevokeStillWorks(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	rotated, rotatedKeys := m.rotate(t, epoch().Add(time.Hour))
	state, err := identity.NewState(m.secondRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	if state, err = state.Adopt(rotated); err != nil || state.KeyEpoch() != 1 {
		t.Fatalf("Adopt() = epoch %d, %v, want the keys of epoch 1", state.KeyEpoch(), err)
	}
	keys, err := state.OpenAccountKeys()
	if err != nil || keys.BoxPublicKey() != rotatedKeys.BoxPublicKey() {
		t.Fatalf("OpenAccountKeys() = %v", err)
	}
	if _, _, err := rotated.RevokeDevice(m.first, keys, m.second.Public().Sign, identity.Lost, rand.Reader, epoch().Add(2*time.Hour)); err != nil {
		t.Errorf("RevokeDevice() after a rotation = %v", err)
	}
}

// thiefChains returns the chain where the stolen laptop adds a device of the thief and revokes the phone, the chain just after the thief
// added its device, and the chain where the owner revoked the stolen laptop with the words at the same place.
func (m team) thiefChains(tb testing.TB) (added, thief, owner identity.Chain, attacker identity.DeviceKey) {
	tb.Helper()
	attacker = newDeviceOf(tb)
	added, err := appended(m.chain, m.first, attacker.Public(), "Thief phone", epoch().Add(2*time.Hour))
	if err != nil {
		tb.Fatal(err)
	}
	thief, _, err = added.RevokeDevice(m.first, m.firstKeys, m.second.Public().Sign, identity.Lost, rand.Reader, epoch().Add(3*time.Hour))
	if err != nil {
		tb.Fatal(err)
	}
	owner, _, err = m.chain.RevokeDevice(m.recovery, m.keys, m.first.Public().Sign, identity.Revocation{Reason: identity.ReasonStolen, Since: epoch().Add(time.Hour)}, rand.Reader, epoch().Add(4*time.Hour))
	if err != nil {
		tb.Fatal(err)
	}
	return added, thief, owner, attacker
}

func TestOutranks_AChainOfTheRecoveryKeyReplacesTheEntriesOfADeviceAtAFork(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, thief, owner, _ := m.thiefChains(t)
	if !owner.Outranks(thief) || thief.Outranks(owner) {
		t.Errorf("owner outranks thief = %v, thief outranks owner = %v, want true and false", owner.Outranks(thief), thief.Outranks(owner))
	}
}

func TestOutranks_IsFalseWithoutAForkOrForTwoDevices(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if chain.Outranks(m.chain) || m.chain.Outranks(chain) || chain.Outranks(chain) {
		t.Error("a chain outranks its own prefix, extension or itself")
	}
	a, err := appended(m.chain, m.first, newDeviceOf(t).Public(), "Tablet", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	b, err := appended(m.chain, m.second, newDeviceOf(t).Public(), "Desktop", epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if a.Outranks(b) || b.Outranks(a) {
		t.Error("a fork of two devices has a winner")
	}
}

func TestOutranks_TwoRecoveryEntriesHaveExactlyOneWinner(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	a, _ := m.rotate(t, epoch().Add(time.Hour))
	b, _, err := m.chain.RevokeDevice(m.recovery, m.keys, m.second.Public().Sign, identity.Lost, rand.Reader, epoch().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if a.Outranks(b) == b.Outranks(a) {
		t.Errorf("a outranks b = %v, b outranks a = %v, want exactly one", a.Outranks(b), b.Outranks(a))
	}
}

func TestAdopt_AThiefWithOneDeviceCannotKeepTheOtherDevicesRevoked(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	added, thief, owner, attacker := m.thiefChains(t)
	phone, err := identity.NewState(m.secondRecord(t), m.chain)
	if err != nil {
		t.Fatal(err)
	}
	if phone, err = phone.Adopt(thief); err != nil || phone.Status != identity.StatusRevoked {
		t.Fatalf("the thief revoked the phone: status = %v, error = %v, want revoked", phone.Status, err)
	}
	phone, err = phone.Adopt(owner)
	if err != nil || phone.Status != identity.StatusActive {
		t.Fatalf("the words of the owner: status = %v, error = %v, want the phone active again", phone.Status, err)
	}
	keys, err := phone.OpenAccountKeys()
	if err != nil || keys.Epoch() != 1 || keys.BoxPublicKey() != owner.AccountBoxKey() {
		t.Errorf("the phone keys: epoch = %d, error = %v, want the keys of the revoke of the owner", keys.Epoch(), err)
	}
	laptop, err := identity.NewState(m.firstRecord(t), thief)
	if err != nil {
		t.Fatal(err)
	}
	if laptop, err = laptop.Adopt(owner); err != nil || laptop.Status != identity.StatusRevoked {
		t.Errorf("the stolen laptop: status = %v, error = %v, want revoked", laptop.Status, err)
	}
	wrapped, err := m.keys.WrapForDevice(attacker.Public())
	if err != nil {
		t.Fatal(err)
	}
	thiefPhone, err := identity.NewState(identity.Record{Device: attacker, Wrapped: wrapped}, added)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := thiefPhone.Adopt(owner); !errors.Is(err, identity.ErrDeviceVoided) {
		t.Errorf("the device the thief added: error = %v, want ErrDeviceVoided", err)
	}
}

func TestAdopt_AChainOfTwoDevicesDoesNotReplaceTheChainOfTheState(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	_, thief, _, _ := m.thiefChains(t)
	state, err := identity.NewState(m.firstRecord(t), thief)
	if err != nil {
		t.Fatal(err)
	}
	other, err := appended(m.chain, m.second, newDeviceOf(t).Public(), "Desktop", epoch().Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := state.Adopt(other); !errors.Is(err, identity.ErrNotExtension) {
		t.Errorf("error = %v, want ErrNotExtension", err)
	}
}

func TestChain_ContainsTheMarkItHadBeforeItGrew(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	mark := m.chain.GetMark()
	chain, _ := m.revokeSecond(t)
	if !chain.Contains(mark) || !m.chain.Contains(mark) {
		t.Errorf("a chain that grew or stayed does not contain mark %v", mark)
	}
	_, thief, owner, _ := m.thiefChains(t)
	if !thief.Contains(mark) || !owner.Contains(mark) {
		t.Error("a fork after the mark does not contain it")
	}
}

func TestChain_DoesNotContainAMarkOfALongerOrOtherHistory(t *testing.T) {
	t.Parallel()
	m := newTeam(t)
	chain, _ := m.revokeSecond(t)
	if m.chain.Contains(chain.GetMark()) {
		t.Error("a shorter chain contains the mark of a longer one")
	}
	other := newTeam(t)
	if other.chain.Contains(m.chain.GetMark()) {
		t.Error("the chain of another account contains the mark")
	}
	if m.chain.Contains(identity.Mark{}) {
		t.Error("a chain contains the empty mark")
	}
}

func TestParseMark_ReadsWhatMarkWrote(t *testing.T) {
	t.Parallel()
	mark := newTeam(t).chain.GetMark()
	parsed, err := identity.ParseMark(" " + mark.String()[:2] + " " + mark.String()[2:] + " ")
	if err != nil || parsed != mark {
		t.Errorf("ParseMark(%q) = %v, %v, want %v", mark.String(), parsed, err, mark)
	}
}

func TestParseMark_RefusesWhatIsNotAMark(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"", "3", "3-", "0-AABBCCDDEEFF", "x-AABBCCDDEEFF", "3-AABB", "3-AABBCCDDEEFF00", "3-ZZBBCCDDEEFF"} {
		if _, err := identity.ParseMark(text); !errors.Is(err, identity.ErrMalformed) {
			t.Errorf("ParseMark(%q) error = %v, want ErrMalformed", text, err)
		}
	}
}
